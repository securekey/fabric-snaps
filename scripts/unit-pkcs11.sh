#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e

PKC11_TOOL=github.com/gbolo/go-util/p11tool
GO_SRC=/opt/gopath/src

#Add entry here below for your key to be imported into softhsm
declare -a PRIVATE_KEYS=(
    "github.com/securekey/fabric-snaps/httpsnap/cmd/sampleconfig/ec-keys/client.key"
    "github.com/securekey/fabric-snaps/transactionsnap/cmd/sampleconfig/msp/keystore/key.pem"
)

# list all modules requiring PKCS11 testing
declare -a PKG_TESTS=(
    "github.com/securekey/fabric-snaps/httpsnap"
    "github.com/securekey/fabric-snaps/transactionsnap"
)

echo "Installing pkcs11 tool..."
go get ${PKC11_TOOL}

echo "Importing keys to softhsm..."
softhsm2-util --init-token --slot 1 --label "ForFabric" --pin 98765432 --so-pin 987654

cd ${GO_SRC}/${PKC11_TOOL}
for i in "${PRIVATE_KEYS[@]}"
do
    echo "Importing key : ${GO_SRC}/${i}"
    openssl pkcs8 -topk8 -inform PEM -outform PEM -nocrypt -in ${GO_SRC}/${i} -out private.p8
    go run main.go -action import -keyFile private.p8
    rm -rf private.p8
done


echo "Running PKCS11 unit tests..."

PKGS=""
for i in "${PKG_TESTS[@]}"
do
    PKGS_LIST=`go list "${i}"... 2> /dev/null | \
                    grep -v /api`
    PKGS+=" $PKGS_LIST"
done

go test -count=1 -tags pkcs11 -cover $PKGS -p 1 -timeout=10m
