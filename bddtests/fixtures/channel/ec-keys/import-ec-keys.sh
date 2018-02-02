#!/usr/bin/env bash

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
#    "ec-keys/httpsnap-abc-client.key"
#    "ec-keys/httpsnap-test-client.key"
#    "ec-keys/p1_sk"
#    "ec-keys/p2_sk"
#    "ec-keys/server.key"
#    "ec-keys/server-1.key"
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

