#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e

#TODO below commented lines will be replaced with proper key import steps once EC keys import available in softhsm image
#cd /go/src/github.com/securekey/fabric-snaps/httpsnap/cmd/sampleconfig/msp/keystore/
#openssl pkcs8 -topk8 -inform PEM -outform PEM -nocrypt -in test-client.key -out private.p8
#softhsm2-util --import private.p8 --slot 1069122796  --label "SKLogs" --pin 98765432  --id A1B2 --no-public-key
#rm -f private.p8


PKGS=`go list github.com/securekey/fabric-snaps/httpsnap... 2> /dev/null | \
                                                 grep -v /api`
echo "Running PKCS11 unit tests..."
go test -tags pkcs11 -cover $PKGS -p 1 -timeout=10m
