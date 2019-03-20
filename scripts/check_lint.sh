#!/bin/bash

#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# This script runs Go linting and vetting tools

set -e

runLint() {
    dirs=$*
    for i in `echo $dirs`
    do
        if [ -d "${i}" ]
        then
            go_files_count=`find ${i} -name "*.go" | wc -l`
            if [ ${go_files_count} -gt 0 ]
            then
                echo "Running lint for directory ${i}..."
                golangci-lint -v run ${i}/... -c .golangci.yml
                echo "Linting done for ${i}."
            fi
        fi
    done
}

FS_DIR=$1

echo "FS_DIR=${FS_DIR}"

#apk add --update alpine-sdk
#apk add git gcc libstdc++

#apt-get update
#apt-get install -y libtool libltdl-dev

export GOPROXY=
go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

cd "${FS_DIR}"

export GO111MODULE=on

go env

echo "GO111MODULE=${GO111MODULE}"
echo "GOPROXY=${GOPROXY}"

env

runLint "."



