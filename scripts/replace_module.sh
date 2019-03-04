#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# replace modules in fabric-snaps go.mod
sed 's/\github.com\/securekey\/fabric-next.*/..\//g' -i $GOPATH/src/github.com/securekey/fabric-snaps/go.mod
sed 's/\github.com\/securekey\/fabric-snaps/github.com\/hyperledger\/fabric\/plugins/g' -i $GOPATH/src/github.com/securekey/fabric-snaps/go.mod
# replace modules in rolesmgr go.mod
sed 's/\github.com\/securekey\/fabric-next.*/..\/..\/..\/..\//g' -i $GOPATH/src/github.com/securekey/fabric-snaps/util/rolesmgr/go.mod
sed 's/\github.com\/securekey\/fabric-snaps/github.com\/hyperledger\/fabric\/plugins\/util\/rolesmgr/g' -i $GOPATH/src/github.com/securekey/fabric-snaps/util/rolesmgr/go.mod
# replace modules in statemgr go.mod
sed 's/\github.com\/securekey\/fabric-next.*/..\/..\/..\/..\//g' -i $GOPATH/src/github.com/securekey/fabric-snaps/util/statemgr/go.mod
sed 's/\github.com\/securekey\/fabric-snaps/github.com\/hyperledger\/fabric\/plugins\/util\/statemgr/g' -i $GOPATH/src/github.com/securekey/fabric-snaps/util/statemgr/go.mod