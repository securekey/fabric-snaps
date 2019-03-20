#!/bin/bash

#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# This script runs Go linting and vetting tools

set -e
GOLANGCI_LINT_CMD=golangci-lint

FS_DIR=$1
TMP_MOD=${GO111MODULE}
TMP_GOPATH=${GOPATH}

cd "${FS_DIR}"

BUILD_TMP=`mktemp -d 2>/dev/null || mktemp -d -t 'fabricsnaps'`
export GO111MODULE=off
export GOPATH=${BUILD_TMP}
go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
export GOPATH=${TMP_GOPATH}
mkdir -p ${GOPATH}/bin
cp -f ${BUILD_TMP}/bin/${GOLANGCI_LINT_CMD} ${GOPATH}/bin/
rm -rf "${BUILD_TMP}"

export GO111MODULE=on

${GOLANGCI_LINT_CMD} run ./... -c .golangci.yml

export GO111MODULE=${TMP_MOD}