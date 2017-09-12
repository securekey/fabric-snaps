#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

go build -ldflags "-linkmode external -extldflags '-static'" -o /opt/snapsbinary/httpsnap /opt/gopath/src/github.com/securekey/fabric-snaps/httpsnap/cmd/httpsnap.go
go build -ldflags "-linkmode external -extldflags '-static'" -o /opt/snapsbinary/transactionsnap  /opt/gopath/src/github.com/securekey/fabric-snaps/transactionsnap/cmd/transactionsnap.go
peer chaincode package -n httpsnap -p /opt/snapsbinary/httpsnap -v $FABRIC_VERSION /opt/snaps/httpsnap.golang -l binary
peer chaincode package -n txnsnap -p /opt/snapsbinary/transactionsnap -v $FABRIC_VERSION /opt/snaps/txnsnap.golang -l binary
/bin/chmod 775 /opt/snaps/*
