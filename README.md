# Fabric Snaps

A Snap is a fabric extension that implements the Chaincode Shim interface. It is deployed as a system chaincode plugin.
The snaps are located at the root of this project. Each snap forms an independently deployable shared object binary.

##### Quick Start

Install:
- A Git Client
- Go - 1.10 or later
- Docker - 17.06.2-ce or later
- Docker Compose - 1.14.0 or later
- You may need libtool - sudo apt-get install -y libtool (linux) or brew install libtool (macOS)
- You may need GNU tar on macOS -  brew install gnu-tar --with-default-names

```
$ cd $GOPATH/src/github.com/securekey/fabric-snaps
$ make integration-test
```

##### Configure
Each snap relies on the Configuration Snap to read configuration from the ledger. The configuration for each application(or snap) is done either on a per-MSP basis or MSP-App basis.
A ConfigCLI tool is provided to update, delete or query these application configurations. Please refer to the [docs](configurationsnap/cmd/configcli/README.md) for sample usage of this tool.

Each snap contains contains a sample configuration directory. For example, `transactionsnap/cmd/sampleconfig`. This directory will contain default a `yaml` configuration file and other configurations that the application may need.

Here is a sample configuration JSON that may be provided to the CLI tool to configure the snaps in this repository:
```
{
    "MspID":"Org1MSP",
    "Peers":[{
        "PeerID":"peer0.org1.example.com",
        "App":[{
            "AppName":"txnsnap",
            "Config":"file://./transactionsnap/cmd/sampleconfig/config.yaml",
             "Version": "1"
        },
        {
            "AppName":"httpsnap",
            "Config":"file://./httpsnap/cmd/sampleconfig/config.yaml",
             "Version": "1"
        },
        {
            "AppName":"eventsnap",
            "Config":"file://./eventsnap/cmd/sampleconfig/configch1.yaml",
             "Version": "1"
        }]
    }
  ]
}
```
Here is a sample of configuration JSON that may be provided to the CLI tool to configure the snaps for peer-less config:
```
{
  "MspID": "Org1MSP",
  "Apps": [
    {
      "AppName": "app1",
      "Version": "1",
      "Config": "{config goes here}"
    },
    {
      "AppName": "app2",
      "Version": "1",
      "Config": "{and config for app2 goes here}"
    }
  ]
}
```
Here is a sample of configuration JSON that may be provided to the CLI tool to configure the snaps for peer-less config and apps with components:
```
{
  "MspID": "General",
  "Apps": [
    {
      "AppName": "auditpolicies",
      "Version": "1",
      "Components": [
        {
          "Name": "sk-td",
          "Config": "{type}"
        },
        {
          "Name": "sk-bmo",
          "Config": "{type}"
        }
      ]
    }
  ]


```




##### Build
The snaps present in this project currently depend on a custom version of fabric located at https://github.com/securekey/fabric-next
This version of fabric contains certain features that have been cherry-picked, a dynamic build to enable Go plugins, and the 'experimental' build tag set. Please see the README located in the project mentioned above for instructions on how to build this.

To build snaps:
```
$ cd $GOPATH/src/github.com/securekey/fabric-snaps
$ make snaps
```
This will produce deployable plugins at build/snaps

##### Deploy
To deploy a system chaincode plugin you must define it in the `chaincode.systemPlugins` section in fabric's `core.yaml` configuration file and whitelist it in the `chaincode.system` section. As an example, you can refer to the test configuration file located at `bddtests/fixtures/config/fabric/core.yaml` lines 458 and 476:
```
chaincode
  system:
    txnsnap: enable
  systemPlugins:
    - enabled: true
      name: txnsnap
      path: /opt/extsysccs/txnsnap.so
      invokableExternal: true
      invokableCC2CC: true
```

To mount the snaps and configuration files into the container we use docker volumes. Example, `bddtests/fixtures/docker-compose.yml` lines 66-69:
```
    - ../../build/snaps/transactionsnap.so:/opt/extsysccs/transactionsnap.so
    - ./config/snaps/:/opt/extsysccs/config
```

##### Test

In order to run fabric-snaps tests against your local environment:
 - Modify properties in `bddtests/fixtures/clientconfig/config.yaml` to point to specific environment. This file includes channel configurations, TLS certs, hosts, ports, MSP(crypto config) directories, timeouts, and security configurations.
 - Run all fabric-snaps tests in `fabric-snaps/bddtests` using the following command `DISABLE_COMPOSITION=true go test`

Run specific tests using the following commands:
```
// Smoke tests
$ DISABLE_COMPOSITION=true go test -run smoke
// HTTP Snap tests
$ DISABLE_COMPOSITION=true go test -run httpsnap
// Transaction Snap tests
$ DISABLE_COMPOSITION=true go test -run txnsnap
```

Test pre-requisites:          
 - Pre-enrolled admin and user for the specified environment. These certs are read from the MSP Directory whose path is defined by the key `client.cryptoconfig.path` in the config.yaml file
 - Ability to create channel “mychannel”. To generate custom channel config blocks and transactions, use the make target `make channel-artifacts`. Channel and org names can be configured in the script `scripts/generate_channeltx.sh`
 - Ability to deploy two chaincodes example_cc and httpsnaptest_cc
 - External Connectivity for HttpSnap
