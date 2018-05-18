/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package querycmd

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/action"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/cliconfig"
	"github.com/spf13/cobra"
)

const description = `
The query command allows the client to query the org's configuration using a Config Key.
The Config Key consists of:

* MspID (mandatory) - The MSP ID of the organization
* PeerID (optional) - The ID of the peer
* AppName (optional) - The application name
* ConfigVer (optional) - The config version

The Config Key may be specified as a JSON string (using the --configkey option) or it may
be specified using the options: --mspid, --peerid, --appname and --configver.

If PeerID and AppName are not specified then all of the org's configuration is returned.
`

const examples = `
- Query a single peer for configuration of a particular application:
    $ ./configcli query --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --peerurl grpcs://localhost:7051 --mspid Org1MSP --peerid peer0.org1.example.com --appname myapp --configver 1

... results in the following output:

    --------------------------------------------------------------------
    ----- MSPID: Org1MSP, Peer: peer0.org1.example.com, App: myapp:
    embedded config
    --------------------------------------------------------------------

- To display the output in raw format:
    $ ./configcli query --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --peerurl grpcs://localhost:7051 --mspid Org1MSP --peerid peer0.org1.example.com --appname myapp --configver 1 --format raw

... results in the following output (note that this string would need to be unmarshalled using json.Unmarshal in order to get a readable config Value):

    [{"Key":{"MspID":"Org1MSP","PeerID":"peer0.org1.example.com","AppName":"myapp","Version":"1"},"Value":"ZW1iZWRkZWQgY29uZmln"}]

- Query a single peer for all configuration for Org1MSP:
    $ ./configcli query --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --peerurl grpcs://localhost:7051 --mspid Org1MSP

- Query a single peer using a config key:
    $ ./configcli query --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --peerurl grpcs://localhost:7051 --configkey '{"MspID":"Org1MSP","PeerID":"peer0.org1.example.com","AppName":"app1","Version":"1"}'
`

// Cmd returns the Query command
func Cmd() *cobra.Command {
	return newCmd(action.New())
}

type queryAction struct {
	action.Action
}

func newCmd(baseAction action.Action) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "query",
		Short:   "Query configuration",
		Long:    description,
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			action, err := newQueryAction(baseAction)
			if err != nil {
				return errors.Wrap(err, "error while initializing queryAction")
			}
			if len(action.Peers()) == 0 {
				return errors.New("please specify an orgid, mspid, or a peer to connect to")
			}
			return action.query()
		},
	}

	flags := cmd.Flags()

	cliconfig.InitPeerURL(flags)
	cliconfig.InitChannelID(flags)
	cliconfig.InitConfigKey(flags)
	cliconfig.InitPeerID(flags)
	cliconfig.InitAppName(flags)
	cliconfig.InitOutputFormat(flags)
	cliconfig.InitConfigVer(flags)

	return cmd
}

func newQueryAction(baseAction action.Action) (*queryAction, error) {
	action := &queryAction{
		Action: baseAction,
	}
	err := action.Initialize()
	return action, err
}

func (a *queryAction) query() error {
	key, err := a.ConfigKey()
	if err != nil {
		return err
	}

	if key.MspID == "" {
		return errors.New("invalid config key: MspID not specified")
	}

	configKeyBytes, err := json.Marshal(key)
	if err != nil {
		return errors.Wrapf(err, "error marshalling config key")
	}

	cliconfig.Config().Logger().Debugf("Using config key: %s\n", configKeyBytes)

	response, err := a.Query(cliconfig.ConfigSnapID, "get", [][]byte{[]byte(configKeyBytes)})
	if err != nil {
		return err
	}

	Print(response)

	return nil
}
