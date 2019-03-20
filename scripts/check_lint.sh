#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# This script runs Go linting and vetting tools

set -e

LINT_CMD=golangci-lint


function finish {
  rm -rf vendor
}
trap finish EXIT


echo "Running linters..."
${LINT_CMD} run ./... -c ".golangci.yml"
