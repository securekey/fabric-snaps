#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# This script runs Go linting and vetting tools

set -e


GOMETALINT_CMD=gometalinter


function finish {
  rm -rf vendor
}
trap finish EXIT


echo "Running metalinters..."
go mod vendor
GO111MODULE=off $GOMETALINT_CMD --config=./gometalinter.json ./...
