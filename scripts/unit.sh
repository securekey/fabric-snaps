#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e


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


# Packages to exclude
PKGS=`go list github.com/hyperledger/fabric/plugins/... 2> /dev/null | \
                                                 grep -v /build | \
                                                 grep -v /vendor/ | \
                                                 grep -v /mocks | \
                                                 grep -v /api | \
                                                 grep -v /protos | \
                                                 grep -v /scripts/fabric-sdk-go | \
                                                 grep -v /bddtests`
echo "Running tests..."
go test -count=1 -tags "testing" -cover $PKGS -p 1 -timeout=10m
