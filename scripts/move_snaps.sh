#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e

# These dependencies will have their own instance with a different version than the one in fabric
# Not all libraries will support this.
declare -a FLATTEN_EXCEPTIONS=("github.com/spf13/viper" "github.com/spf13/pflag" "github.com/spf13/cast" "github.com/magiconair/properties")

echo "Creating temp directories within fabric..."
mkdir -p /opt/gopath/src/github.com/hyperledger/fabric/plugins/
mkdir -p /opt/gopath/src/github.com/hyperledger/fabric/plugins/bddtests/fixtures/snapexample

echo "Copying snaps to subdirectory within fabric..."
cp -r /opt/gopath/src/github.com/securekey/fabric-snaps/* /opt/gopath/src/github.com/hyperledger/fabric/plugins/

echo "Rewriting import paths..."
find /opt/gopath/src/github.com/hyperledger/fabric/plugins -type f -name "*.go" -print0 | xargs -0 sed -i "s/github.com\/securekey\/fabric-snaps\//github.com\/hyperledger\/fabric\/plugins\//g"

echo "Backing up exceptions (deps excluded from the flattening script)..."
mkdir -p vendor_backup/temp
for dep in "${FLATTEN_EXCEPTIONS[@]}"
do
  mkdir -p vendor_backup/${dep}
  cp -r ./vendor/${dep}/* vendor_backup/${dep}/
done

echo "Flattening dependencies..."
./scripts/flatten_deps.sh /opt/gopath/src/github.com/hyperledger/fabric ./vendor
find ./vendor -name '*_test.go' -delete

echo "Deleting dependencies that have a mismatch with fabric..."
# This is required because govendor allows different versions of subpackages while godep does not
rm -rf ./vendor/golang.org/x/crypto/sha3/
rm -rf ./vendor/golang.org/x/net/context/
rm -rf ./vendor/golang.org/x/sys/unix/

## remove when update to fabric 1.3
rm -rf ./vendor/github.com/golang/protobuf/
rm -rf ./vendor/github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos
rm -rf ./vendor/github.com/hyperledger/fabric-sdk-go/internal/github.com/hyperledger/fabric/protos
rm -rf ./vendor/github.com/hyperledger/fabric-sdk-go/internal/github.com/hyperledger/fabric/discovery
rm -rf ./vendor/github.com/hyperledger/fabric-sdk-go/pkg/fab/discovery
cp -r /opt/gopath/src/github.com/hyperledger/fabric/plugins/scripts/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos ./vendor/github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/
cp -r /opt/gopath/src/github.com/hyperledger/fabric/plugins/scripts/fabric-sdk-go/internal/github.com/hyperledger/fabric/protos ./vendor/github.com/hyperledger/fabric-sdk-go/internal/github.com/hyperledger/fabric/
cp -r /opt/gopath/src/github.com/hyperledger/fabric/plugins/scripts/fabric-sdk-go/internal/github.com/hyperledger/fabric/discovery ./vendor/github.com/hyperledger/fabric-sdk-go/internal/github.com/hyperledger/fabric/
cp -r /opt/gopath/src/github.com/hyperledger/fabric/plugins/scripts/fabric-sdk-go/pkg/fab/discovery ./vendor/github.com/hyperledger/fabric-sdk-go/pkg/fab/


echo "Restoring exceptions"
cp -r ./vendor_backup/* ./vendor/

echo "Copying flattened vendor to subdirectory within fabric..."
rm -rf /opt/gopath/src/github.com/hyperledger/fabric/plugins/vendor
cp -r ./vendor /opt/gopath/src/github.com/hyperledger/fabric/plugins/

rm -rf /opt/gopath/src/github.com/securekey/fabric-snaps
