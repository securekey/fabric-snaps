#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e

# Packages to exclude
PKGS=`go list github.com/securekey/fabric-snaps/... 2> /dev/null | \
                                                 grep -v /build | \
                                                 grep -v /vendor/ | \
                                                 grep -v /mocks | \
                                                 grep -v /api | \
                                                 grep -v /protos | \
                                                 grep -v /bddtests`
echo "Running tests..."
go test -tags "testing" -cover $PKGS -p 1 -timeout=10m
