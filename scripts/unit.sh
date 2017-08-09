#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e

# Packages to exclude
PKGS=`go list github.com/securekey/fabric-snaps/... 2> /dev/null | \
                                                 grep -v /vendor/ | \
                                                 grep -v /mocks | \
                                                 grep -v /api | \
                                                 grep -v /protos | \
                                                 grep -v /bddtests`
echo "Running tests..."
gocov test -ldflags "$GO_LDFLAGS" $PKGS -p 1 -timeout=5m | gocov-xml > report.xml
