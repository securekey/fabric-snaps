#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0


set -e
GO_CMD="${GO_CMD:-go}"
LINT_CMD="${LINT_CMD:-golangci-lint}"
GOPATH="${GOPATH:-${HOME}/go}"

function installGoLangCi {
    declare repo="github.com/golangci/golangci-lint/cmd/golangci-lint"
    declare revision="v1.15.0"
    installGoPkg "${repo}" "${revision}" "" "golangci-lint"
}

function installGoGas {
    declare repo="github.com/GoASTScanner/gas"
    declare revision="4ae8c95"
    GO111MODULE=off GOPATH=${BUILD_TMP} ${GO_CMD} get -u github.com/kisielk/gotool
    GO111MODULE=off GOPATH=${BUILD_TMP} ${GO_CMD} get -u github.com/nbutton23/zxcvbn-go
    GO111MODULE=off GOPATH=${BUILD_TMP} ${GO_CMD} get -u github.com/ryanuber/go-glob
    GO111MODULE=off GOPATH=${BUILD_TMP} ${GO_CMD} get -u gopkg.in/yaml.v2
    installGoPkg "${repo}" "${revision}" "/cmd/gas/..." "gas"
}

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

function installDependencies {
    echo "Installing dependencies ..."
    BUILD_TMP=`mktemp -d 2>/dev/null || mktemp -d -t 'fabricsnaps'`
    GO111MODULE=off GOPATH=${BUILD_TMP} ${GO_CMD} get -u github.com/axw/gocov/...
    GO111MODULE=off GOPATH=${BUILD_TMP} ${GO_CMD} get -u github.com/AlekSi/gocov-xml
    GO111MODULE=off GOPATH=${BUILD_TMP} ${GO_CMD} get -u github.com/golang/mock/mockgen
    GO111MODULE=off GOPATH=${BUILD_TMP} ${GO_CMD} get -u github.com/client9/misspell/cmd/misspell
    GO111MODULE=off GOPATH=${BUILD_TMP} ${GO_CMD} get -u golang.org/x/tools/cmd/goimports
    installGoLangCi
    # gas in gometalinter is out of date.
    installGoGas
    rm -Rf ${BUILD_TMP}
}

installDependencies