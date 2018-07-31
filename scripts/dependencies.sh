#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# This script installs dependencies for testing tools
# Environment variables that affect this script:
# GO_DEP_COMMIT: Tag or commit level of the go dep tool to install

set -e

GO_CMD="${GO_CMD:-go}"
GO_DEP_CMD="${GO_DEP_CMD:-dep}"
GO_DEP_REPO="github.com/golang/dep"
GO_METALINTER_CMD="${GO_METALINTER_CMD:-gometalinter}"
GOPATH="${GOPATH:-${HOME}/go}"

function installGoDep {
    declare repo=$1
    declare revision=$2

    installGoPkg "${repo}" "${revision}" "/cmd/dep" "dep"
}

function installGoMetalinter {
    declare repo="github.com/alecthomas/gometalinter"
    declare revision=""

    declare pkg="github.com/alecthomas/gometalinter"

    installGoPkg "${repo}" "${revision}" "" "gometalinter"

    rm -Rf ${GOPATH}/src/${pkg}
    mkdir -p ${GOPATH}/src/${pkg}
    cp -Rf ${BUILD_TMP}/src/${repo}/* ${GOPATH}/src/${pkg}/
    ${GO_METALINTER_CMD} --install --force
}

function installGoGas {
    declare repo="github.com/GoASTScanner/gas"
    declare revision="4ae8c95"

    GOPATH=${BUILD_TMP} ${GO_CMD} get -u github.com/kisielk/gotool
    GOPATH=${BUILD_TMP} ${GO_CMD} get -u github.com/nbutton23/zxcvbn-go
    GOPATH=${BUILD_TMP} ${GO_CMD} get -u github.com/ryanuber/go-glob
    GOPATH=${BUILD_TMP} ${GO_CMD} get -u gopkg.in/yaml.v2

    installGoPkg "${repo}" "${revision}" "/cmd/gas/..." "gas"
}

function installGoPkg {
    declare repo=$1
    declare revision=$2
    declare pkgPath=$3
    shift 3
    declare -a cmds=$@

    echo "Installing ${repo}@${revision} to $GOPATH/bin ..."

    GOPATH=${BUILD_TMP} go get -d ${repo}
    tag=$(cd ${BUILD_TMP}/src/${repo} && git tag -l | sort -V --reverse | head -n 1 | grep "${revision}" || true)
    if [ ! -z "${tag}" ]; then
        revision=${tag}
        echo "  using tag ${revision}"
    fi
    (cd ${BUILD_TMP}/src/${repo} && git reset --hard ${revision})
    GOPATH=${BUILD_TMP} go install -i ${repo}/${pkgPath}

}

function installDependencies {
    echo "Installing dependencies ..."

    BUILD_TMP=`mktemp -d 2>/dev/null || mktemp -d -t 'fabricsnaps'`
    GOPATH=${BUILD_TMP} ${GO_CMD} get -u github.com/axw/gocov/...
    GOPATH=${BUILD_TMP} ${GO_CMD} get -u github.com/AlekSi/gocov-xml
    GOPATH=${BUILD_TMP} ${GO_CMD} get -u github.com/golang/mock/mockgen

    installGoMetalinter

    # gas in gometalinter is out of date.
    installGoGas

    # Install specific version of go dep (particularly for CI)
    if [ -n "${GO_DEP_COMMIT}" ]; then
        installGoDep ${GO_DEP_REPO} ${GO_DEP_COMMIT}
    fi
    rm -Rf ${BUILD_TMP}

}

installDependencies
