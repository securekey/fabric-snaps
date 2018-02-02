#!/usr/bin/env bash

#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e

#This is temporary script which will import peer msp key to HSM so that peer can start successfully
#TODO This script should be removed once required keys are already available in peer HSM

PKC11_TOOL=github.com/gbolo/go-util/p11tool
GO_SRC=/opt/gopath/src

#Add entry here below for your key to be imported into softhsm
declare -a PRIVATE_KEYS=(
    "/etc/hyperledger/msp/peer/keystore/a23272f898441edb0f415f028b3b3b42627d75e150dfbae23543234751814983_sk"
)


echo "Installing pkcs11 tool..."
go get ${PKC11_TOOL}

echo "Importing keys to softhsm..."
softhsm2-util --init-token --slot 1 --label "ForFabric" --pin 98765432 --so-pin 987654

cd ${GO_SRC}/${PKC11_TOOL}
for i in "${PRIVATE_KEYS[@]}"
do
    echo "Importing key : ${GO_SRC}/${i}"
    openssl pkcs8 -topk8 -inform PEM -outform PEM -nocrypt -in ${i} -out private.p8
    go run main.go -action import -keyFile private.p8
    rm -rf private.p8
done

