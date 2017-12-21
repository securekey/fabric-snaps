#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

set -e

mkdir -p /opt/gopath/src/github.com/hyperledger
mkdir -p /opt/gopath/src/github.com/securekey

cp -r /opt/temp/src/github.com/securekey/fabric-snaps /opt/gopath/src/github.com/securekey

echo "Cloning fabric..."
cd /opt/gopath/src/github.com/hyperledger
git clone https://github.com/securekey/fabric-next.git
cd fabric-next/scripts
git checkout $FABRIC_NEXT_VERSION
./fabric_cherry_picks.sh >/dev/null
cd /opt/gopath/src/github.com/hyperledger/fabric
git apply /opt/gopath/src/github.com/hyperledger/fabric-next/patches/peerCLITLS.patch

cd  /opt/gopath/src/github.com/securekey/fabric-snaps

# This can be deleted once we have Go 1.10
echo "Patching Go Compiler..."
patch -p1 $GOROOT/src/cmd/link/internal/ld/data.go ./scripts/patches/linker.patch
echo "Re-building Go Compiler..."
go install -a cmd

echo "Executing move script..."
./scripts/move_snaps.sh

cd /opt/gopath/src/github.com/hyperledger/fabric/plugins

echo "Building plugins..."
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./httpsnap.so github.com/hyperledger/fabric/plugins/httpsnap/cmd
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./transactionsnap.so github.com/hyperledger/fabric/plugins/transactionsnap/cmd
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./membershipsnap.so github.com/hyperledger/fabric/plugins/membershipsnap/cmd
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./eventsnap.so github.com/hyperledger/fabric/plugins/eventsnap/cmd
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./txnsnapinvoker.so github.com/hyperledger/fabric/plugins/bddtests/fixtures/snapexample/txnsnapinvoker
go build -tags $GO_BUILD_TAGS -buildmode=plugin -ldflags "-X github.com/hyperledger/fabric/plugins/configurationsnap/cmd/configurationscc/configdata.PublicKeyForLogging=SAMPLE-KEY-1234 -X github.com/hyperledger/fabric/plugins/configurationsnap/cmd/configurationscc/configdata.KeyIDForLogging=SAMPLE-KEYID-1234"  -o ./configurationscc.so github.com/hyperledger/fabric/plugins/configurationsnap/cmd/configurationscc
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./eventconsumersnap.so github.com/hyperledger/fabric/plugins/bddtests/fixtures/snapexample/eventconsumersnap


cp httpsnap.so /opt/temp/src/github.com/securekey/fabric-snaps/build/snaps/
cp transactionsnap.so /opt/temp/src/github.com/securekey/fabric-snaps/build/snaps/
cp membershipsnap.so /opt/temp/src/github.com/securekey/fabric-snaps/build/snaps/
cp eventsnap.so /opt/temp/src/github.com/securekey/fabric-snaps/build/snaps/
cp txnsnapinvoker.so /opt/temp/src/github.com/securekey/fabric-snaps/build/test/
cp configurationscc.so /opt/temp/src/github.com/securekey/fabric-snaps/build/snaps/
cp eventconsumersnap.so /opt/temp/src/github.com/securekey/fabric-snaps/build/test/
