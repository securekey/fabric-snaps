#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e

mkdir -p $GOPATH/src/github.com/securekey
cp -r /tmp/securekey/fabric-snaps $GOPATH/src/github.com/securekey


export GO111MODULE=on

$CONFIG_GIT

#Add entry here below for your key to be imported into softhsm
declare -a PRIVATE_KEYS=(
    "github.com/securekey/fabric-snaps/httpsnap/cmd/sampleconfig/ec-keys/client.key"
    "github.com/securekey/fabric-snaps/transactionsnap/cmd/sampleconfig/msp/keystore/key.pem"
)

# list all modules requiring PKCS11 testing
declare PKG_TESTS="github.com/securekey/fabric-snaps/httpsnap/... github.com/securekey/fabric-snaps/transactionsnap/..."




echo "Importing keys to softhsm..."

for i in "${PRIVATE_KEYS[@]}"
do
    echo "Importing key : ${GOPATH}/src/${i}"
    openssl pkcs8 -topk8 -inform PEM -outform PEM -nocrypt -in ${GOPATH}/src/${i} -out private.p8
    pkcs11helper -action import -keyFile private.p8
    rm -rf private.p8
done


echo "Running PKCS11 unit tests..."
cd $GOPATH/src/github.com/securekey/fabric-snaps
rm go.sum

GO111MODULE=on go test -count=1 -tags pkcs11 -cover $PKG_TESTS -p 1 -timeout=10m
