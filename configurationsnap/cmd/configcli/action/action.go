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

	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/cryptosuite/bccsp/multisuite"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defcore"
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
		return nil, errors.Errorf(errors.GeneralError, "Error getting channel client: %s", err)
	}

	resp, err := channelClient.Query(
		channel.Request{
			ChaincodeID: chaincodeID,
			Fcn:         fctn,
			Args:        args,
		},
		channel.WithTargets(a.peers...),
		channel.WithRetry(retry.DefaultChannelOpts),
	)
	if err != nil {
		return nil, err
	}
	return resp.Payload, nil
}

// ExecuteTx executes a transaction on the given chaincode with the given function and args
func (a *action) ExecuteTx(chaincodeID, fctn string, args [][]byte) error {
	channelClient, err := a.ChannelClient()
	if err != nil {
		return errors.Errorf(errors.GeneralError, "Error getting channel client: %s", err)
	}
	_, err = channelClient.Execute(
		channel.Request{
			ChaincodeID: chaincodeID,
			Fcn:         fctn,
			Args:        args,
		},
		channel.WithTargets(a.peers...),
		channel.WithRetry(retry.DefaultChannelOpts),
	)

	return err
}

// ConfigKey resolves a ConfigKey from the command-line arguments
func (a *action) ConfigKey() (*mgmtapi.ConfigKey, error) {
	if cliconfig.Config().ConfigKey() != "" {
		queryBytes := []byte(cliconfig.Config().ConfigKey())
		configKey, err := configkeyutil.Unmarshal(queryBytes)
		if err != nil {
			return nil, errors.Errorf(errors.GeneralError, "invalid config key: %s", err)
		}
		return configKey, nil
	}

	mspID := cliconfig.Config().GetMspID()
	if mspID == "" {
		// MSP ID not provide. Attempt to get the MSP ID from the Org ID
		orgID := a.OrgID()
		if orgID != "" {
			orgConfig, ok := cliconfig.Config().NetworkConfig().Organizations[orgID]
			if !ok {
				return nil, errors.Errorf(errors.GeneralError, "org config not found for org [%s]", orgID)
			}
			mspID = orgConfig.MSPID
			cliconfig.Config().Logger().Debugf("Attempted to get MspID from org [%s]. MspID [%s]\n", orgID, mspID)
		}
	}

	return &mgmtapi.ConfigKey{
		MspID:            mspID,
		PeerID:           cliconfig.Config().PeerID(),
		AppName:          cliconfig.Config().AppName(),
		AppVersion:       cliconfig.Config().AppVer(),
		ComponentName:    cliconfig.Config().ComponentName(),
		ComponentVersion: cliconfig.Config().ComponentVer(),
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

	sdk, err := fabsdk.New(nil,
		fabsdk.WithEndpointConfig(cliconfig.Config()),
		fabsdk.WithIdentityConfig(cliconfig.Config()),
		fabsdk.WithCryptoSuiteConfig(cliconfig.Config()),
		fabsdk.WithCorePkg(&customCorePkg{}),
	)
	if err != nil {
		return errors.Errorf(errors.GeneralError, "Error initializing SDK: %s", err)
	}
	a.sdk = sdk
	return nil
}

func (a *action) initTargetPeers() error {
	netConfig := cliconfig.Config().NetworkConfig()

	selectedOrgID := cliconfig.Config().OrgID()
	if selectedOrgID == "" {
		selectedOrgID = cliconfig.Config().Client().Organization
	}

	cliconfig.Config().Logger().Debugf("Selected org [%s]\n", selectedOrgID)

	a.orgIDByPeer = make(map[string]string)

	for orgID := range netConfig.Organizations {
		if err := a.initTargetPeersForOrg(orgID, selectedOrgID); err != nil {
			return err
		}
	}

	cliconfig.Config().Logger().Debugf("All peers: %+v\n", a.peers)

	return nil
}

func (a *action) initTargetPeersForOrg(orgID, selectedOrgID string) error {
	cliconfig.Config().Logger().Debugf("Getting peers for org [%s]\n", orgID)

	peersConfig, ok := cliconfig.Config().PeersConfig(orgID)
	if !ok {
		return errors.Errorf(errors.GeneralError, "peer config not found for org [%s]", orgID)
	}

	orgConfig, ok := cliconfig.Config().NetworkConfig().Organizations[orgID]
	if !ok {
		return errors.Errorf(errors.GeneralError, "org config not found for org [%s]", orgID)
	}

	cliconfig.Config().Logger().Debugf("Peers for org [%s]: %+v\n", orgID, peersConfig)

	for _, p := range peersConfig {
		if a.includePeer(orgConfig.MSPID, p, orgID, selectedOrgID) {
			cliconfig.Config().Logger().Debugf("Adding peer for org [%s]: %s\n", orgID, p.URL)

			endorser, err := peer.New(cliconfig.Config(), peer.FromPeerConfig(&fabApi.NetworkPeer{PeerConfig: p, MSPID: orgConfig.MSPID}))
			if err != nil {
				return errors.Wrap(errors.GeneralError, err, "NewPeer return error")
			}

			a.peers = append(a.peers, endorser)
			a.orgIDByPeer[endorser.URL()] = orgID
		}
	}
	return nil
}

func (a *action) includePeer(orgMSPID string, peerConfig fabApi.PeerConfig, orgID, selectedOrgID string) bool {
	// See if the peer is blacklisted
	_, ok := cliconfig.Config().PeerConfig(peerConfig.URL)
	if !ok {
		return false
	}

	if len(cliconfig.Config().Peers()) > 0 {
		return contains(cliconfig.Config().Peers(), peerConfig.URL)
	}

	// An org ID and/or MSP ID was specified. Include if the peer's org/MSP matches
	return (selectedOrgID == orgID || cliconfig.Config().GetMspID() == orgMSPID)
}

func contains(vals []string, val string) bool {
	for _, value := range vals {
		if value == val {
			return true
		}
	}
	return false
}

func readFromTerminal(prompt string, responsech chan string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s :", prompt)
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

//customCorePkg to use mutlisuite cryptosuite impl to support both SW and PKCS11 on demand
type customCorePkg struct {
	defcore.ProviderFactory
}

// CreateCryptoSuiteProvider returns a implementation of factory default bccsp cryptosuite
func (f *customCorePkg) CreateCryptoSuiteProvider(config core.CryptoSuiteConfig) (core.CryptoSuite, error) {
	cryptoSuiteProvider, err := multisuite.GetSuiteByConfig(config)
	return cryptoSuiteProvider, err
}
