#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0


set -e
GO_CMD="${GO_CMD:-go}"
GOLANGCI_LINT_CMD="${GO_METALINTER_CMD:-golangci-lint}"
GOPATH="${GOPATH:-${HOME}/go}"


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

    declare repo="github.com/golangci/golangci-lint/cmd/golangci-lint"
    declare revision="v1.15.0"
    installGoPkg "${repo}" "${revision}" "" "${GOLANGCI_LINT_CMD}"
}

function installDependencies {
    echo "Installing dependencies ..."
    BUILD_TMP=`mktemp -d 2>/dev/null || mktemp -d -t 'fabricsnaps'`
    GO111MODULE=off GOPATH=${BUILD_TMP} ${GO_CMD} get -u github.com/axw/gocov/...
    GO111MODULE=off GOPATH=${BUILD_TMP} ${GO_CMD} get -u github.com/AlekSi/gocov-xml
    GO111MODULE=off GOPATH=${BUILD_TMP} ${GO_CMD} get -u github.com/golang/mock/mockgen

    installGolangCi

    rm -Rf ${BUILD_TMP}
}

installDependencies
