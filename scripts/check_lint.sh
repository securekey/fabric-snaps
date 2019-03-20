#!/bin/bash

#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# This script runs Go linting and vetting tools

set -e
GO_CMD="${GO_CMD:-go}"
GOPATH="${GOPATH:-${HOME}/go}"
GOLANGCI_LINT_CMD=golangci-lint

function installGoPkg {
    declare repo=$1
    declare revision=$2
    declare pkgPath=$3
    shift 3
    declare -a cmds=$@
    echo "Installing ${repo}@${revision} to $GOPATH/bin ..."
    GO111MODULE=off GOPATH=${BUILD_TMP} go get -d ${repo}
    tag=$(cd ${BUILD_TMP}/src/${repo} && git tag -l | sort -V --reverse | head -n 1 | grep "${revision}" || true)
    if [ ! -z "${tag}" ]; then
        revision=${tag}
        echo "  using tag ${revision}"
    fi
    (cd ${BUILD_TMP}/src/${repo} && git reset --hard ${revision})
    echo " Checking $GOPATH ..."
    GO111MODULE=off GOPATH=${BUILD_TMP} go install -i ${repo}/${pkgPath}
    mkdir -p ${GOPATH}/bin
    for cmd in ${cmds[@]}
    do
        echo "Copying ${cmd} to ${GOPATH}/bin"
        cp -f ${BUILD_TMP}/bin/${cmd} ${GOPATH}/bin/
    done
}

function installGolangCi {
    echo "Installing ${GOLANGCI_LINT_CMD} ..."
    BUILD_TMP=`mktemp -d 2>/dev/null || mktemp -d -t 'fabricsnaps'`

    declare repo="github.com/golangci/golangci-lint/cmd/golangci-lint"
    declare revision="v1.15.0"
    installGoPkg "${repo}" "${revision}" "" "${GOLANGCI_LINT_CMD}"

    rm -Rf ${BUILD_TMP}
}

installGolangCi

TMP_GOPROXY=${GOPROXY}
NEW_GOPROXY="https://athens:Na5ZcpmKjPM7XZTW@eng-athens.onetap.ca"

if [ "${GOHOSTARCH}" == "s390x" ]
then
    export GOPROXY="${NEW_GOPROXY}"
    echo "exporting GOPROXY; newValue=${GOPROXY} ; oldValue=${TMP_GOPROXY}"
fi

echo "GOPROXY=${GOPROXY}"

GO111MODULE=on ${GOLANGCI_LINT_CMD} -v run ./... -c .golangci.yml


if [ "${GOHOSTARCH}" == "s390x" ]
then
    export GOPROXY="${TMP_GOPROXY}"
    echo "exporting back GOPROXY; newValue=${GOPROXY} ; oldValue=${NEW_GOPROXY}"
fi

