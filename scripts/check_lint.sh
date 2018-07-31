#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# This script runs Go linting and vetting tools

set -e

set -e


GOMETALINT_CMD=gometalinter


echo "Running metalinters..."
$GOMETALINT_CMD --config=./gometalinter.json ./...
