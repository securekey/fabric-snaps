# Fabric Snaps

A Snap is a fabric extension that implements the Chaincode Shim interface. It is deployed as a process inside the peer container.
The snaps are located at the root of this project. Each snap forms an independently deployable executable.

##### Configure
Each snap contains contains a sample configuration directory. For example, `transactionsnap/cmd/sampleconfig`. This directory will contain default a `yaml` configuration file as well as any other configurations that the snap requires.

##### Build
Note: We assume a working Golang and Docker installation.

The snaps present in this project currently depend on a custom version of fabric located at https://github.com/securekey/fabric-next

As a prerequisite, you must clone and build this project first:
```
$ cd $GOPATH/src/github.com/hyperledger/fabric
$ git clone https://github.com/securekey/fabric-next.git fabric
$ cd fabric
// Checkout the the tag corresponding to this project
$ git checkout v17.08.02
$ make docker
```

To build snaps:
```
$ cd $GOPATH/src/github.com/securekey/fabric-snaps
$ make snaps
```
This will produce deployable artifacts at build/snaps

##### Deploy
To deploy a snap, the peer is configured with environment variables. As an example, you can refer to the docker-compose file located at `bddtests/fixtures/docker-compose.yml` lines 82-85 and 92-97:
```
      # enable External SCCs
      - CORE_CHAINCODE_SYSTEMEXT_ENABLED=true
      # path of External SCCs to read CodeSpec objects
      - CORE_CHAINCODE_SYSTEMEXT_CDS_PATH=/opt/extsysccs

      # Txn Snap
      - CORE_CHAINCODE_SYSTEMEXT_TXNSNAP_ENABLED=true
      - CORE_CHAINCODE_SYSTEMEXT_TXNSNAP_EXECENV=SYSTEM_EXT
      - CORE_CHAINCODE_SYSTEMEXT_TXNSNAP_INVOKABLEEXTERNAL=true
      - CORE_CHAINCODE_SYSTEMEXT_TXNSNAP_INVOKABLECC2CC=true
      - CORE_CHAINCODE_SYSTEMEXT_TXNSNAP_CONFIGPATH=/opt/snaps/txnsnap
```

To mount the snaps and configuration files into the container we use docker volumes. Example, lines 115-116:
```
    - ./config/extsysccs:/opt/extsysccs
    - ./config/snaps/:/opt/snaps
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

