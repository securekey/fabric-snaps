#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

sed 's/\gerrit.securekey.com\/fabric-mod.*/..\//g' -i /go/src/github.com/securekey/fabric-snaps/go.mod;sed 's/\github.com\/securekey\/fabric-snaps/github.com\/hyperledger\/fabric\/plugins/g' -i /go/src/github.com/securekey/fabric-snaps/go.mod
sed 's/\github.com\/securekey\/fabric-kevlar\/fsblkstorage.*/..\/fabric-kevlar\/fsblkstorage/g' -i /go/src/github.com/securekey/fabric-snaps/go.mod
sed 's/\gerrit.securekey.com\/fabric-mod.*/..\/..\/..\/..\//g' -i /go/src/github.com/securekey/fabric-snaps/util/rolesmgr/go.mod;sed 's/\github.com\/securekey\/fabric-snaps/github.com\/hyperledger\/fabric\/plugins\/util\/rolesmgr/g' -i /go/src/github.com/securekey/fabric-snaps/util/rolesmgr/go.mod
sed 's/\gerrit.securekey.com\/fabric-mod.*/..\/..\/..\/..\//g' -i /go/src/github.com/securekey/fabric-snaps/util/statemgr/go.mod;sed 's/\github.com\/securekey\/fabric-snaps/github.com\/hyperledger\/fabric\/plugins\/util\/statemgr/g' -i /go/src/github.com/securekey/fabric-snaps/util/statemgr/go.mod

