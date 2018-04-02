/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package action

import (
	"fmt"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/peer"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/cliconfig"
	"github.com/spf13/pflag"
)

// MockInvoker allows mock implementation for the ExecuteTx and Query functions
type MockInvoker func(chaincodeID, fctn string, args [][]byte) ([]byte, error)

// MockAction provides a mock implementation of Action
type MockAction struct {
	action
	Invoker  MockInvoker
	Response []byte
}

// Initialize initializes the action
func (a *MockAction) Initialize() error {
	if err := cliconfig.InitConfig(); err != nil {
		return err
	}

	logging.SetLevel("", levelFromName(cliconfig.Config().LoggingLevel()))

	channelID := cliconfig.Config().ChannelID()
	if channelID == "" {
		return errors.New("no channel ID specified")
	}

	return a.initTargetPeers()
}

// ChannelClient creates a new channel client
func (a *MockAction) ChannelClient() (*channel.Client, error) {
	panic("not implemented")
}

// Query queries the given chaincode with the given function and args and returns a response
func (a *MockAction) Query(chaincodeID, fctn string, args [][]byte) ([]byte, error) {
	return a.Invoker(chaincodeID, fctn, args)
}

// ExecuteTx executes a transaction on the given chaincode with the given function and args
func (a *MockAction) ExecuteTx(chaincodeID, fctn string, args [][]byte) error {
	_, err := a.Invoker(chaincodeID, fctn, args)
	return err
}

// InitGlobalFlags initializes the global command flags
func InitGlobalFlags(flags *pflag.FlagSet) {
	cliconfig.InitLoggingLevel(flags)
	cliconfig.InitClientConfigFile(flags)
	cliconfig.InitChannelID(flags)
	cliconfig.InitUserName(flags)
	cliconfig.InitUserPassword(flags)
	cliconfig.InitOrgID(flags)
	cliconfig.InitMspID(flags)
	cliconfig.InitKeyType(flags)
	cliconfig.InitEphemeralFlag(flags)
	cliconfig.InitSigAlg(flags)
}

// NewMockPeer creates a mock peer
func NewMockPeer(url string, mspID string) fabApi.Peer {
	config := mocks.NewMockEndpointConfig()
	peer, err := peer.New(config, peer.WithURL(url), peer.WithMSPID(mspID))
	if err != nil {
		panic(fmt.Sprintf("Failed to create peer: %v)", err))
	}
	return peer
}
