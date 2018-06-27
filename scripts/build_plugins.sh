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


cd  /opt/gopath/src/github.com/securekey/fabric-snaps


echo "Executing move script..."
./scripts/move_snaps.sh

cd /opt/gopath/src/github.com/hyperledger/fabric/plugins

echo "Building plugins..."
echo "Building transaction snap..."
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./transactionsnap.so github.com/hyperledger/fabric/plugins/transactionsnap/cmd
echo "Building http snap..."
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./httpsnap.so github.com/hyperledger/fabric/plugins/httpsnap/cmd
echo "Building membership snap..."
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./membershipsnap.so github.com/hyperledger/fabric/plugins/membershipsnap/cmd
echo "Building event snap..."
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./eventsnap.so github.com/hyperledger/fabric/plugins/eventsnap/cmd
echo "Building txn snap invoker..."
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./txnsnapinvoker.so github.com/hyperledger/fabric/plugins/bddtests/fixtures/snapexample/txnsnapinvoker
echo "Building configuration snap..."
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./configurationscc.so github.com/hyperledger/fabric/plugins/configurationsnap/cmd/configurationscc
echo "Building eventconsumer snap..."
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./eventconsumersnap.so github.com/hyperledger/fabric/plugins/bddtests/fixtures/snapexample/eventconsumersnap
echo "Building bootstrap snap..."
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./bootstrapsnap.so github.com/hyperledger/fabric/plugins/bddtests/fixtures/snapexample/bootstrap
echo "Building metrics filter..."
go build -tags "$GO_BUILD_TAGS" -buildmode=plugin -o ./metricsfilter.so github.com/hyperledger/fabric/plugins/metrics/cmd/filter

cp httpsnap.so /opt/temp/src/github.com/securekey/fabric-snaps/build/snaps/
cp transactionsnap.so /opt/temp/src/github.com/securekey/fabric-snaps/build/snaps/
cp membershipsnap.so /opt/temp/src/github.com/securekey/fabric-snaps/build/snaps/
cp eventsnap.so /opt/temp/src/github.com/securekey/fabric-snaps/build/snaps/
cp txnsnapinvoker.so /opt/temp/src/github.com/securekey/fabric-snaps/build/test/
cp configurationscc.so /opt/temp/src/github.com/securekey/fabric-snaps/build/snaps/
cp eventconsumersnap.so /opt/temp/src/github.com/securekey/fabric-snaps/build/test/
cp bootstrapsnap.so /opt/temp/src/github.com/securekey/fabric-snaps/build/test/
