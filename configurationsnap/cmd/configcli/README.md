# Config CLI

Config CLI is a command-line interface that allows an organization to manage application configuration which is stored in the ledger. It provides the following functionality:

- Create or update the configuration of one or more applications within an organization (MSP)
- Query the configuration of one or more applications within an organization
- Delete configuration

## Commands

The Config CLI provides three command: update, query, and delete.

### update

The update command allows a client to update the configuration of one or more applications. Configuration may be specified direcly on the command-line as a JSON string (using the --config option) or a configuration file may be specified (using the --configfile option).

The format of the configuration is as follows:

{
  "MspID": "msp.one",
  "Peers": [
    {
      "PeerID": "peer1",
      "App": [
        {
          "AppName": "app1",
          "Version": "1",
          "Config": "config for app1"

        },
        {
          "AppName": "app1",
          "Version": "2",
          "Config": "config for app1 v2"

        },
        {
          "AppName": "app2",
          "Version": "1",
          "Config": "file://path_to_config.yaml"
        }
      ]
    },
    {
      "PeerID":"peer2",
      . . .
	}
  ]
}

The configuration may be embedded direcly in the "Config" element or the Config element may reference a file containing the configuration. 

### query

The query command allows the client to query the org's configuration using a Config Key. The Config Key consists of:

* MspID (mandatory) - The MSP ID of the organization
* PeerID (optional) - The ID of the peer
* AppName (optional) - The application name

The Config Key may be specified as a JSON string (using the --configkey option) or it may be specified using the options: --mspid, --peerid, and --appname.

If PeerID and AppName are not specified then all of the org's configuration is returned.

### delete

The delete command allows the client to delete the org's configuration using a Config Key. (The config key is the same as described in the Query command.) A specific application configuration may be deleted if PeerID and AppName are specified, or the org's entire configuration may be deleted if only MspID is specified.

## Running

Navigate to folder configurationsnap/cmd/configcli.

$ go build
$ ./configcli <command> [options]

To display the available commands and global options:

$ ./configcli help
$ ./configcli --help

To display the available options for a specific command:

$ ./configcli help <command>
$ ./configcli <command> --help

## Sample Usage

### update

Send the update to all peers within the MSP, "Org1MSP" using a configuration file:

    $ ./configcli update --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --mspid Org1MSP --configfile ./sampleconfig/org1-config.json

Send the update to a single peer:

    $ ./configcli update --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --peerurl grpcs://localhost:7051 --configfile ./sampleconfig/org1-config.json

Send an update using a configuration string specified in the command-line:

    $ ./configcli update --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --mspid Org1MSP --config '{"MspID":"Org1MSP","Peers":[{"PeerID":"peer0.org1.example.com","App":[{"AppName":"myapp","Version":"1","Config":"embedded config"}]}]}'

Send an update using a configuration for peer-less config string specified in the command-line:
    $ ./configcli update --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --mspid Org1MSP --config '{"MspID":"Org1MSP", "Apps":[{AppName": "app1", "Version":"1", "Config": "{config goes here}"}]}'

### query

Query a single peer for configuration of a particular application:

    $ ./configcli query --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --peerurl grpcs://localhost:7051 --mspid Org1MSP --peerid peer0.org1.example.com --appname myapp

... results in the following output:

    --------------------------------------------------------------------
    ----- MSPID: Org1MSP, Peer: peer0.org1.example.com, App: myapp:
    embedded config
    --------------------------------------------------------------------

To display the output in raw format:

    $ ./configcli query --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --peerurl grpcs://localhost:7051 --mspid Org1MSP --peerid peer0.org1.example.com --appname myapp --format raw

... results in the following output (note that this string would need to be unmarshalled using json.Unmarshal in order to get a readable config Value):

    [{"Key":{"MspID":"Org1MSP","PeerID":"peer0.org1.example.com","AppName":"myapp"},"Value":"ZW1iZWRkZWQgY29uZmln"}]

Query a single peer for all configuration for Org1MSP:

    $ ./configcli query --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --peerurl grpcs://localhost:7051 --mspid Org1MSP

Query a single peer using a config key:

    $ ./configcli query --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --peerurl grpcs://localhost:7051 --configkey '{"MspID":"Org1MSP","PeerID":"peer0.org1.example.com","AppName":"app1"}'

### delete

Delete a the configuration of a particular application:

    $ ./configcli delete --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --mspid Org1MSP --peerid peer0.org1.example.com --appname myapp

Delete all configuration in Org1MSP:

    $ ./configcli delete --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --mspid Org1MSP
