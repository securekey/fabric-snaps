#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e

# Packages to exclude
PKGS=`go list github.com/securekey/fabric-snaps/bddtests/... 2> /dev/null | \
                                                 grep -v /fixtures | \
                                                 grep -v /vendor`
echo "Running integration tests..."
go test -count=1 -v -cover $PKGS -p 1 -timeout=20m