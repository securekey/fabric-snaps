#!/bin/bash

#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# This script runs Go linting and vetting tools

set -e

#apk add --no-cache coreutils

mkdir -p $GOPATH/src/github.com/hyperledger
mkdir -p $GOPATH/src/github.com/securekey

cp -r /opt/temp/src/github.com/securekey/fabric-snaps $GOPATH/src/github.com/securekey
cp /opt/temp/src/github.com/securekey/fabric-snaps/.golangci.yml $GOPATH/src/github.com/securekey/fabric-snaps/.golangci.yml
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
cd $GOPATH/src/github.com/hyperledger/fabric/plugins
./scripts/replace_module.sh

make depend

export GOCACHE=""

GO_CMD="${GO_CMD:-go}"
GOPATH="${GOPATH:-${HOME}/go}"
GOLANGCI_LINT_CMD="${GOLANGCI_LINT_CMD:-golangci-lint}"

TMP_GOPROXY=${GOPROXY}

if [ -n "${PROXY_FOR_GOLANGCI}" ]
then
    export GOPROXY="${PROXY_FOR_GOLANGCI}"
    echo "exporting GOPROXY; newValue=${GOPROXY} ; oldValue=${TMP_GOPROXY}"
fi

echo "Before running ${GOLANGCI_LINT_CMD}: GOPROXY=${GOPROXY}"
gofmt -e -d -s -w ./
GO111MODULE=on ${GOLANGCI_LINT_CMD} -v run ./... -c .golangci.yml

if [ -n "${PROXY_FOR_GOLANGCI}" ]
then
    export GOPROXY="${TMP_GOPROXY}"
    echo "exporting back GOPROXY; newValue=${GOPROXY} ; oldValue=${NEW_GOPROXY}"
fi

echo "After running ${GOLANGCI_LINT_CMD}: GOPROXY=${GOPROXY}"
