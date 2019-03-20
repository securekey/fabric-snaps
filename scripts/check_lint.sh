#!/bin/bash

#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# This script runs Go linting and vetting tools

set -e

FS_DIR=$1
export GOPROXY=
go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
export GO111MODULE=on

cd "${FS_DIR}"
golangci-lint -v run ./... -c .golangci.yml

