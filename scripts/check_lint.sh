#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# This script runs Go linting and vetting tools

set -e

GO_CMD="${GO_CMD:-go}"
LINT_CMD="golangci-lint"

mkdir -p "${GOPATH}"/src/github.com/securekey
cp -R /opt/temp/src/github.com/securekey/fabric-snaps "${GOPATH}"/src/github.com/securekey

cd "${GOPATH}"/src/github.com/securekey/fabric-snaps

apt-get update
apt-get -y install libtool libltdl-dev

export GO111MODULE=on
export GOPROXY=https://athens:Na5ZcpmKjPM7XZTW@eng-athens.onetap.ca

${LINT_CMD} -v run ./... -c ".golangci.yml"
