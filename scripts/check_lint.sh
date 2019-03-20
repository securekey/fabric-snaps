#!/bin/bash

#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# This script runs Go linting and vetting tools


set -e

#apk add --update alpine-sdk
#apk add git gcc libstdc++

apt-get update
apt-get install -y libtool libltdl-dev

go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

cd /go/src/github.com/securekey/fabric-snaps

export GO111MODULE=on

for i in `ls -1 | grep -v "bddtests" | grep -v "build" | grep -v "scripts" | grep -v "images" `
do
    if [ -d "${i}" ]
    then
        echo "Running lint for directory ${i}..."
        golangci-lint -v run ${i}/... -c .golangci.yml
        echo "Linting done for ${i}."
    fi
done


