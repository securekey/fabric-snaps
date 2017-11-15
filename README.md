# Fabric Snaps

A Snap is a fabric extension that implements the Chaincode Shim interface. It is deployed as a system chaincode plugin.
The snaps are located at the root of this project. Each snap forms an independently deployable shared object binary.

##### Configure
(TODO: update this once config moves to the ledger)
Each snap contains contains a sample configuration directory. For example, `transactionsnap/cmd/sampleconfig`. This directory will contain default a `yaml` configuration file as well as any other configurations that the snap requires.

##### Build
Note: We assume a working Golang(v1.9.2) and Docker(v17.09.0-ce) installation.

The snaps present in this project currently depend on a custom version of fabric located at https://github.com/securekey/fabric-next
This version of fabric contains certain features that have been cherry-picked, a dynamic build to enable Go plugins, and the 'experimental' build tag set. Please see the README located in the project mentioned above for instructions on how to build this.

*Note:* The tagged version of fabric-snaps being used must match the corresponding tag in fabric-next. e.g v17.11.1 of fabric-snaps is compatible with v17.11.1 of fabric-next.

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
