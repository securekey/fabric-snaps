#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e


echo "Creating temp directories within fabric..."
mkdir -p $GOPATH/src/github.com/hyperledger/fabric/plugins/

echo "Copying snaps to subdirectory within fabric..."
cp -r $GOPATH/src/github.com/securekey/fabric-snaps/* $GOPATH/src/github.com/hyperledger/fabric/plugins/
cp $GOPATH/src/github.com/securekey/fabric-snaps/.golangci.yml $GOPATH/src/github.com/hyperledger/fabric/plugins/


echo "Rewriting import paths..."
find $GOPATH/src/github.com/hyperledger/fabric/plugins -type f -name "*.*" -print0 | xargs -0 sed -i "s/github.com\/securekey\/fabric-snaps\//github.com\/hyperledger\/fabric\/plugins\//g"

rm -rf $GOPATH/src/github.com/securekey/fabric-snaps
rm -rf $GOPATH/src/github.com/hyperledger/fabric/go.sum
