#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#


sedGoPath=$(echo $GOPATH | sed 's#/#\\/#g')

# replace modules in fabric-snaps go.mod
sed 's/\github.com\/securekey\/fabric-next.*/$GOPATH\/src\/github.com\/hyperledger\/fabric/g' -i $GOPATH/src/github.com/hyperledger/fabric/plugins/go.mod
sed 's/\github.com\/securekey\/fabric-snaps/github.com\/hyperledger\/fabric\/plugins/g' -i $GOPATH/src/github.com/hyperledger/fabric/plugins/go.mod
sed 's/$GOPATH/'"$sedGoPath"'/' -i $GOPATH/src/github.com/hyperledger/fabric/plugins/go.mod

# replace modules in rolesmgr go.mod
sed 's/\github.com\/securekey\/fabric-next.*/$GOPATH\/src\/github.com\/hyperledger\/fabric/g' -i $GOPATH/src/github.com/hyperledger/fabric/plugins/util/rolesmgr/go.mod
sed 's/\github.com\/securekey\/fabric-snaps\/util/github.com\/hyperledger\/fabric\/plugin\/util/g' -i $GOPATH/src/github.com/hyperledger/fabric/plugins/util/rolesmgr/go.mod
sed 's/$GOPATH/'"$sedGoPath"'/' -i $GOPATH/src/github.com/hyperledger/fabric/plugins/util/rolesmgr/go.mod

# replace modules in statemgr go.mod
sed 's/\github.com\/securekey\/fabric-next.*/$GOPATH\/src\/github.com\/hyperledger\/fabric/g' -i $GOPATH/src/github.com/hyperledger/fabric/plugins/util/statemgr/go.mod
sed 's/\github.com\/securekey\/fabric-snaps\/util/github.com\/hyperledger\/fabric\/plugins\/util/g' -i $GOPATH/src/github.com/hyperledger/fabric/plugins/util/statemgr/go.mod
sed 's/$GOPATH/'"$sedGoPath"'/' -i $GOPATH/src/github.com/hyperledger/fabric/plugins/util/statemgr/go.mod
