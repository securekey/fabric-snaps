#!/bin/bash

#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# This script runs Go linting and vetting tools

set -e
GO_CMD="${GO_CMD:-go}"
GOPATH="${GOPATH:-${HOME}/go}"
GOLANGCI_LINT_CMD="${GO_METALINTER_CMD:-golangci-lint}"

TMP_GOPROXY=${GOPROXY}

if [ -n "${PROXY_FOR_GOLANGCI}" ]
then
    export GOPROXY="${PROXY_FOR_GOLANGCI}"
    echo "exporting GOPROXY; newValue=${GOPROXY} ; oldValue=${TMP_GOPROXY}"
fi

echo "Before running ${GOLANGCI_LINT_CMD}: GOPROXY=${GOPROXY}"

GO111MODULE=on ${GOLANGCI_LINT_CMD} -v run ./... -c .golangci.yml

if [ -n "${PROXY_FOR_GOLANGCI}" ]
then
    export GOPROXY="${TMP_GOPROXY}"
    echo "exporting back GOPROXY; newValue=${GOPROXY} ; oldValue=${NEW_GOPROXY}"
fi

echo "After running ${GOLANGCI_LINT_CMD}: GOPROXY=${GOPROXY}"
