#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# Packages to exclude
PKGS=`go list github.com/securekey/fabric-snaps/bddtests/... 2> /dev/null | \
                                                 grep -v /fixtures | \
                                                 grep -v /vendor`
echo "Running integration tests..."
gocov test -ldflags "$GO_LDFLAGS" $PKGS -p 1 -timeout=5m | gocov-xml > integration-report.xml
