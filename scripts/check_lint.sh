#!/bin/bash

#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# This script runs Go linting and vetting tools

set -e

FS_DIR=$1
cd "${FS_DIR}"
go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

golangci-lint run ./... -c .golangci.yml

