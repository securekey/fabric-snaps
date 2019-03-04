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


declare -a PRIVATE_KEYS=()

for file in /etc/hyperledger/msp/peer/keystore/*_sk
do
    PRIVATE_KEYS+=($file)
done


echo "Importing keys to softhsm..."
for i in "${PRIVATE_KEYS[@]}"
do
    echo "Importing key : ${GO_SRC}${i}"
    openssl pkcs8 -topk8 -inform PEM -outform PEM -nocrypt -in ${i} -out private.p8
    pkcs11helper -action import -keyFile private.p8
    rm -rf private.p8
done