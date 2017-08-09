#!/bin/bash

NAME=txnsnap
CDS=$GOPATH/src/github.com/securekey/fabric-snaps/bddtests/fixtures/config/extsysccs/$NAME.golang
SOURCE=github.com/securekey/fabric-snaps/transactionsnap/cmd

$GOPATH/src/github.com/hyperledger/fabric/build/bin/peer chaincode package -n $NAME -p $SOURCE -v 1.0.0 $CDS

chmod 775 $CDS
