#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

go build -ldflags "-linkmode external -extldflags '-static'" -o /opt/snapsbinary/txnsnapinvoker /opt/gopath/src/github.com/securekey/fabric-snaps/bddtests/fixtures/snapexample/txnsnapinvoker/txnsnapinvoker.go

export PATH=/usr/local/go/bin:$PATH
NAME=txnsnapinvoker
CDS=/opt/snaps/$NAME.golang

peer chaincode package -n $NAME -p /opt/snapsbinary/txnsnapinvoker -v 1.1.0-snapshot-51724a4 $CDS -l binary

chmod 775 $CDS
