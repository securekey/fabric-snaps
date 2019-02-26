#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#


export GO111MODULE=on GOCACHE=on
export GOPATH=/opt/gopath

git config --global url."git@github.com:securekey/fabric-kevlar".insteadOf "https://github.com/securekey/fabric-kevlar"
go version
set -e

mkdir -p /opt/gopath/src/github.com/hyperledger
mkdir -p /opt/gopath/src/github.com/securekey

cp -r /opt/temp/src/github.com/securekey/fabric-snaps /opt/gopath/src/github.com/securekey
rm -rf /opt/gopath/src/github.com/securekey/fabric-snaps/go.sum
sed 's/\gerrit.securekey.com\/fabric-mod.*/..\//g' -i /opt/gopath/src/github.com/securekey/fabric-snaps/go.mod;sed 's/\github.com\/securekey\/fabric-snaps/github.com\/hyperledger\/fabric\/plugins/g' -i /opt/gopath/src/github.com/securekey/fabric-snaps/go.mod
sed 's/\github.com\/securekey\/fabric-kevlar\/fsblkstorage.*/..\/fabric-kevlar\/fsblkstorage/g' -i /opt/gopath/src/github.com/securekey/fabric-snaps/go.mod
sed 's/\gerrit.securekey.com\/fabric-mod.*/..\/..\/..\/..\//g' -i /opt/gopath/src/github.com/securekey/fabric-snaps/util/rolesmgr/go.mod;sed 's/\github.com\/securekey\/fabric-snaps/github.com\/hyperledger\/fabric\/plugins\/util\/rolesmgr/g' -i /opt/gopath/src/github.com/securekey/fabric-snaps/util/rolesmgr/go.mod
sed 's/\gerrit.securekey.com\/fabric-mod.*/..\/..\/..\/..\//g' -i /opt/gopath/src/github.com/securekey/fabric-snaps/util/statemgr/go.mod;sed 's/\github.com\/securekey\/fabric-snaps/github.com\/hyperledger\/fabric\/plugins\/util\/statemgr/g' -i /opt/gopath/src/github.com/securekey/fabric-snaps/util/statemgr/go.mod


echo "Cloning fabric..."
cd /opt/gopath/src/github.com/hyperledger
git clone https://gerrit.securekey.com/fabric-kevlar
cd fabric-kevlar
git checkout $FABRIC_NEXT_VERSION
./scripts/fabric_cherry_picks.sh >/dev/null
cd /opt/gopath/src/github.com/hyperledger/fabric



cd  /opt/gopath/src/github.com/securekey/fabric-snaps
echo "Executing move script..."
./scripts/move_snaps.sh

cd /opt/gopath/src/github.com/hyperledger/fabric/plugins

sed 's/\gerrit.securekey.com\/fabric-mod.*/..\/..\//g' -i /opt/gopath/src/github.com/hyperledger/fabric/fabric-kevlar/fsblkstorage/go.mod
sed 's/\github.com\/securekey\/fabric-kevlar\/fsblkstorage/github.com\/hyperledger\/fabric\/fabric-kevlar\/fsblkstorage/g' -i /opt/gopath/src/github.com/hyperledger/fabric/fabric-kevlar/fsblkstorage/go.mod

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
