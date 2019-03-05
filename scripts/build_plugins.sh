#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#


export GO111MODULE=on GOCACHE=on

set -e

mkdir -p $GOPATH/src/github.com/hyperledger
mkdir -p $GOPATH/src/github.com/securekey

cp -r /opt/temp/src/github.com/securekey/fabric-snaps $GOPATH/src/github.com/securekey
rm -rf $GOPATH/src/github.com/securekey/fabric-snaps/go.sum

echo "Cloning fabric..."
cd $GOPATH/src/github.com/hyperledger
git clone $FABRIC_NEXT_REPO fabric-next
cd fabric-next
git checkout $FABRIC_NEXT_VERSION
./scripts/fabric_cherry_picks.sh >/dev/null


cd  $GOPATH/src/github.com/securekey/fabric-snaps
echo "Executing move script..."
./scripts/move_snaps.sh
echo "Executing replace module..."
cd $GOPATH/src/github.com/hyperledger/fabric/plugins
./scripts/replace_module.sh





echo "Building plugins..."
echo "Building transaction snap..."
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./transactionsnap.so github.com/hyperledger/fabric/plugins/transactionsnap/cmd
echo "Building http snap..."
rm -rf on
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./httpsnap.so github.com/hyperledger/fabric/plugins/httpsnap/cmd
echo "Building membership snap..."
rm -rf on
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./membershipsnap.so github.com/hyperledger/fabric/plugins/membershipsnap/cmd
echo "Building txn snap invoker..."
rm -rf on
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./txnsnapinvoker.so github.com/hyperledger/fabric/plugins/bddtests/fixtures/snapexample/txnsnapinvoker
echo "Building configuration snap..."
rm -rf on
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./configurationscc.so github.com/hyperledger/fabric/plugins/configurationsnap/cmd/configurationscc
echo "Building bootstrap snap..."
rm -rf on
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./bootstrapsnap.so github.com/hyperledger/fabric/plugins/bddtests/fixtures/snapexample/bootstrap

cp httpsnap.so /opt/temp/src/github.com/securekey/fabric-snaps/build/snaps/
cp transactionsnap.so /opt/temp/src/github.com/securekey/fabric-snaps/build/snaps/
cp membershipsnap.so /opt/temp/src/github.com/securekey/fabric-snaps/build/snaps/
cp txnsnapinvoker.so /opt/temp/src/github.com/securekey/fabric-snaps/build/test/
cp configurationscc.so /opt/temp/src/github.com/securekey/fabric-snaps/build/snaps/
cp bootstrapsnap.so /opt/temp/src/github.com/securekey/fabric-snaps/build/test/
