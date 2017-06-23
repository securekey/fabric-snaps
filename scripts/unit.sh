#!/bin/bash

set -e

# Packages to exclude
PKGS=`go list github.com/securekey/fabric-snaps/... 2> /dev/null | \
                                                  grep -v /vendor/`
echo "Running tests..."
gocov test -ldflags "$GO_LDFLAGS" $PKGS -p 1 -timeout=5m | gocov-xml > report.xml
