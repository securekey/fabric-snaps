#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e


echo "Creating temp directories within fabric..."
mkdir -p /opt/gopath/src/github.com/hyperledger/fabric/plugins/

echo "Copying snaps to subdirectory within fabric..."
cp -r /opt/gopath/src/github.com/securekey/fabric-snaps/* /opt/gopath/src/github.com/hyperledger/fabric/plugins/

echo "Rewriting import paths..."
find /opt/gopath/src/github.com/hyperledger/fabric/plugins -type f -name "*.go" -print0 | xargs -0 sed -i "s/github.com\/securekey\/fabric-snaps\//github.com\/hyperledger\/fabric\/plugins\//g"

rm -rf /opt/gopath/src/github.com/securekey/fabric-snaps
rm -rf /opt/gopath/src/github.com/hyperledger/fabric/go.sum
