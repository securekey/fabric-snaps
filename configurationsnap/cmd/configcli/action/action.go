/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package action

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	mgmtapi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/cliconfig"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/configkeyutil"
	"github.com/securekey/fabric-snaps/util/errors"
)

// Action defines the common methods for an command action
type Action interface {
	Initialize() error
	ChannelClient() (*channel.Client, error)
	Peers() []fabApi.Peer
	OrgID() string
	Query(chaincodeID, fctn string, args [][]byte) ([]byte, error)
	ExecuteTx(chaincodeID, fctn string, args [][]byte) error
	ConfigKey() (*mgmtapi.ConfigKey, error)
}

// action is the base implementation of the Action interface.
type action struct {
	peers       []fabApi.Peer
	orgIDByPeer map[string]string
	sdk         *fabsdk.FabricSDK
}

// New returns a new Action
func New() Action {
	return &action{}
}

// Initialize initializes the action
func (a *action) Initialize() error {
	if err := cliconfig.InitConfig(); err != nil {
		return err
	}

	if err := a.initSDK(); err != nil {
		return err
	}

	logging.SetLevel("", levelFromName(cliconfig.Config().LoggingLevel()))

	channelID := cliconfig.Config().ChannelID()
	if channelID == "" {
		return errors.New(errors.GeneralError, "no channel ID specified")
	}

	return a.initTargetPeers()
}

// ChannelClient creates a new channel client
func (a *action) ChannelClient() (*channel.Client, error) {
	userName := cliconfig.Config().UserName()

	chClient, err := channel.New(a.sdk.ChannelContext(cliconfig.Config().ChannelID(), fabsdk.WithUser(userName), fabsdk.WithOrg(a.OrgID())))
	if err != nil {
		return nil, errors.Wrapf(errors.GeneralError, err, "failed to create new channel client")
	}
	return chClient, nil
}

// Peers returns the peers
func (a *action) Peers() []fabApi.Peer {
	return a.peers
}

// OrgID returns the organization ID of the first peer in the list of peers
func (a *action) OrgID() string {
	if len(a.Peers()) == 0 {
		// This shouldn't happen since we should already have passed validation
		panic("no peers to choose from!")
	}

	peer := a.Peers()[0]

	orgID, ok := a.orgIDByPeer[peer.URL()]
	if !ok {
		// This shouldn't happen since we should already have passed validation
		panic(fmt.Sprintf("org not found for peer %s", peer.URL()))
	}

	cliconfig.Config().Logger().Debugf("Org of peer [%s]=[%s]", peer.URL(), orgID)
	return orgID
}

// Query queries the given chaincode with the given function and args and returns a response
func (a *action) Query(chaincodeID, fctn string, args [][]byte) ([]byte, error) {
	channelClient, err := a.ChannelClient()
	if err != nil {
		return nil, errors.Errorf(errors.GeneralError, "Error getting channel client: %v", err)
	}

	resp, err := channelClient.Query(channel.Request{
		ChaincodeID: chaincodeID,
		Fcn:         fctn,
		Args:        args,
	}, channel.WithTargets(a.peers...))
	if err != nil {
		return nil, err
	}
	return resp.Payload, nil
}

// ExecuteTx executes a transaction on the given chaincode with the given function and args
func (a *action) ExecuteTx(chaincodeID, fctn string, args [][]byte) error {
	channelClient, err := a.ChannelClient()
	if err != nil {
		return errors.Errorf(errors.GeneralError, "Error getting channel client: %v", err)
	}
	_, err = channelClient.Execute(
		channel.Request{
			ChaincodeID: chaincodeID,
			Fcn:         fctn,
			Args:        args,
		}, channel.WithTargets(a.peers...))

	return err
}

// ConfigKey resolves a ConfigKey from the command-line arguments
func (a *action) ConfigKey() (*mgmtapi.ConfigKey, error) {
	if cliconfig.Config().ConfigKey() != "" {
		queryBytes := []byte(cliconfig.Config().ConfigKey())
		configKey, err := configkeyutil.Unmarshal(queryBytes)
		if err != nil {
			return nil, errors.Errorf(errors.GeneralError, "invalid config key: %v", err)
		}
		return configKey, nil
	}

	mspID := cliconfig.Config().GetMspID()
	if mspID == "" {
		// MSP ID not provide. Attempt to get the MSP ID from the Org ID
		orgID := a.OrgID()
		if orgID != "" {
			var err error
			mspID, err = cliconfig.Config().MSPID(orgID)
			if err != nil {
				return nil, err
			}
			cliconfig.Config().Logger().Debugf("Attempted to get MspID from org [%s]. MspID [%s]\n", orgID, mspID)
		}
	}

	return &mgmtapi.ConfigKey{
		MspID:   mspID,
		PeerID:  cliconfig.Config().PeerID(),
		AppName: cliconfig.Config().AppName(),
	}, nil
}

// YesNoPrompt prompts the user to enter Y/N. If the user enters 'y' then true is returned.
func YesNoPrompt(prompt string, args ...interface{}) bool {
	fmt.Printf("\n"+prompt+" (Y/N) ", args...)
	ackch := make(chan string)
	go readFromTerminal(prompt, ackch)
	ack := <-ackch
	return strings.ToLower(ack) == "y"
}

func (a *action) initSDK() error {
	if cliconfig.Config().UserName() == "" {
		return errors.New(errors.GeneralError, "user must be specified")
	}

	sdk, err := fabsdk.New(config.FromFile(cliconfig.Config().ConfigFile()),
		fabsdk.WithConfigEndpoint(cliconfig.Config()),
	)
	if err != nil {
		return errors.Errorf(errors.GeneralError, "Error initializing SDK: %s", err)
	}
	a.sdk = sdk
	return nil
}

func (a *action) initTargetPeers() error {
	netConfig, err := cliconfig.Config().NetworkConfig()
	if err != nil {
		return err
	}

	selectedOrgID := cliconfig.Config().OrgID()
	if selectedOrgID == "" {
		selectedOrgID = netConfig.Client.Organization
	}

	cliconfig.Config().Logger().Debugf("Selected org [%s]\n", selectedOrgID)

	a.orgIDByPeer = make(map[string]string)

	for orgID := range netConfig.Organizations {
		cliconfig.Config().Logger().Debugf("Getting peers for org [%s]\n", orgID)

		peersConfig, err := cliconfig.Config().PeersConfig(orgID)
		if err != nil {
			return errors.Wrapf(errors.GeneralError, err, "error getting peer configs for org [%s]", orgID)
		}

		mspID, err := cliconfig.Config().MSPID(orgID)
		if err != nil {
			return errors.Wrapf(errors.GeneralError, err, "error getting MSP ID for org [%s]", orgID)
		}

		cliconfig.Config().Logger().Debugf("Peers for org [%s]: %v\n", orgID, peersConfig)

		for _, p := range peersConfig {

			includePeer := false
			if cliconfig.Config().PeerURL() != "" {
				// A single peer URL was specified. Only include the peer that matches.
				includePeer = cliconfig.Config().PeerURL() == p.URL
			} else {
				// An org ID and/or MSP ID was specified. Include if the peer's org/MSP matches
				includePeer = (selectedOrgID == orgID || cliconfig.Config().GetMspID() == mspID)
			}

			if includePeer {
				cliconfig.Config().Logger().Debugf("Adding peer for org [%s]: %v\n", orgID, p.URL)

				endorser, err := peer.New(cliconfig.Config(), peer.FromPeerConfig(&fabApi.NetworkPeer{PeerConfig: p, MSPID: mspID}))
				if err != nil {
					return errors.Wrap(errors.GeneralError, err, "NewPeer return error")
				}

				a.peers = append(a.peers, endorser)
				a.orgIDByPeer[endorser.URL()] = orgID
			}
		}
	}

	cliconfig.Config().Logger().Debugf("All peers: %v\n", a.peers)

	return nil
}

func readFromTerminal(prompt string, responsech chan string) {
	reader := bufio.NewReader(os.Stdin)
	if response, err := reader.ReadString('\n'); err != nil {
		cliconfig.Config().Logger().Errorf("Error reading from terminal: %s\n", err)
	} else {
		responsech <- response[0:1]
	}
}

func levelFromName(levelName string) logging.Level {
	switch levelName {
	case "ERROR":
		return logging.ERROR
	case "WARNING":
		return logging.WARNING
	case "INFO":
		return logging.INFO
	case "DEBUG":
		return logging.DEBUG
	default:
		return logging.ERROR
	}
}
