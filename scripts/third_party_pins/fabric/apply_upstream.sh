#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# This script fetches code used in the SDK originating from other Hyperledger Fabric projects
# These files are checked into internal paths.
# Note: This script must be adjusted as upstream makes adjustments

set -e

UPSTREAM_PROJECT="github.com/hyperledger/fabric"
UPSTREAM_BRANCH="master"
SCRIPTS_PATH="scripts/third_party_pins/fabric"
PATCHES_PATH="${SCRIPTS_PATH}/patches"

THIRDPARTY_FABRIC_PATH='third_party/github.com/hyperledger/fabric'
THIRDPARTY_FABRIC_API_PATH=$THIRDPARTY_FABRIC_PATH
THIRDPARTY_FABRIC_BCCSP_PKG_PATH=$THIRDPARTY_FABRIC_PATH
THIRDPARTY_INTERNAL_FABRIC_PATH='internal/github.com/hyperledger/fabric'

####
# Clone and patch packages into repo

# Clone original project into temporary directory
echo "Fetching upstream project ($UPSTREAM_PROJECT:$UPSTREAM_COMMIT) ..."
CWD=`pwd`
TMP=`mktemp -d 2>/dev/null || mktemp -d -t 'mytmpdir'`

TMP_PROJECT_PATH=$TMP/src/$UPSTREAM_PROJECT
mkdir -p $TMP_PROJECT_PATH
cd ${TMP_PROJECT_PATH}/..

git clone https://${UPSTREAM_PROJECT}.git
cd $TMP_PROJECT_PATH
git checkout $UPSTREAM_BRANCH
git reset --hard $UPSTREAM_COMMIT

echo "Patching upstream project ..."
git am ${CWD}/${PATCHES_PATH}/*

cd $CWD

# fabric client utils
echo "Pinning and patching fabric client utils..."
declare -a CLIENT_UTILS_IMPORT_SUBSTS=(
    's/[[:space:]]logging[[:space:]]\"github.com/\"github.com/g'
    's/\"github.com\/pkg\/errors/\"github.com\/hyperledger\/fabric-sdk-go\/pkg\/errors/g'
    's/\"github.com\/hyperledger\/fabric\/common\/flogging/flogging\"github.com\/securekey\/fabric-snaps\/internal\/github.com\/hyperledger\/fabric\/sdkpatch\/logbridge/g'
    's/\"github.com\/op\/go-logging/logging\"github.com\/securekey\/fabric-snaps\/internal\/github.com\/hyperledger\/fabric\/sdkpatch\/logbridge/g'
    's/\"github.com\/hyperledger\/fabric\/bccsp/\"github.com\/hyperledger\/fabric-sdk-go\/third_party\/github.com\/hyperledger\/fabric\/bccsp/g'
    's/\"github.com\/hyperledger\/fabric\/common\/cauthdsl/\"github.com\/hyperledger\/fabric-sdk-go\/third_party\/github.com\/hyperledger\/fabric\/common\/cauthdsl/g'
    's/\"github.com\/hyperledger\/fabric\/protos\/utils/\"github.com\/securekey\/fabric-snaps\/internal\/github.com\/hyperledger\/fabric\/protos\/utils/g'
    's/\"github.com\/hyperledger\/fabric\/protos\/common/\"github.com\/hyperledger\/fabric-sdk-go\/third_party\/github.com\/hyperledger\/fabric\/protos\/common/g'
    's/\"github.com\/hyperledger\/fabric\/protos\/peer/\"github.com\/hyperledger\/fabric-sdk-go\/third_party\/github.com\/hyperledger\/fabric\/protos\/peer/g'
    's/\"github.com\/hyperledger\/fabric\/protos\/msp/\"github.com\/hyperledger\/fabric-sdk-go\/third_party\/github.com\/hyperledger\/fabric\/protos\/msp/g'
    's/\"github.com\/hyperledger\/fabric\/protos\/orderer/\"github.com\/hyperledger\/fabric-sdk-go\/third_party\/github.com\/hyperledger\/fabric\/protos\/orderer/g'
    's/\"github.com\/hyperledger\/fabric\/protos\/ledger/\queryresult\"github.com\/securekey\/fabric-snaps\/internal\/github.com\/hyperledger\/fabric\/protos\/ledger/g'
    's/\"github.com\/hyperledger\/fabric\/protos\/ledger/\"github.com\/hyperledger\/fabric-sdk-go\/third_party\/github.com\/hyperledger\/fabric\/protos\/ledger/g'
    's/\"github.com\/hyperledger\/fabric\/protos/\"github.com\/hyperledger\/fabric-sdk-go\/third_party\/github.com\/hyperledger\/fabric\/protos/g'
    's/\"github.com\/hyperledger\/fabric\//\"github.com\/securekey\/fabric-snaps\/internal\/github.com\/hyperledger\/fabric\//g'
)
eval "INTERNAL_PATH=$THIRDPARTY_INTERNAL_FABRIC_PATH TMP_PROJECT_PATH=$TMP_PROJECT_PATH IMPORT_SUBSTS=\"${CLIENT_UTILS_IMPORT_SUBSTS[*]}\" $SCRIPTS_PATH/apply_fabric_client_utils.sh"

# proto utils
echo "Pinning and patching protos (internal) ..."
declare -a PROTOS_INTERNAL_IMPORT_SUBSTS=(
    's/\"github.com\/pkg\/errors/\"github.com\/hyperledger\/fabric-sdk-go\/pkg\/errors/g'
    's/\"github.com\/hyperledger\/fabric\/common\/flogging/flogging\"github.com\/securekey\/fabric-snaps\/internal\/github.com\/hyperledger\/fabric\/sdkpatch\/logbridge/g'
    's/\"github.com\/hyperledger\/fabric\/bccsp/\"github.com\/hyperledger\/fabric-sdk-go\/third_party\/github.com\/hyperledger\/fabric\/bccsp/g'
    's/\"github.com\/hyperledger\/fabric\/protos\/common/\"github.com\/hyperledger\/fabric-sdk-go\/third_party\/github.com\/hyperledger\/fabric\/protos\/common/g'
    's/\"github.com\/hyperledger\/fabric\/protos\/peer/\"github.com\/hyperledger\/fabric-sdk-go\/third_party\/github.com\/hyperledger\/fabric\/protos\/peer/g'
    's/\"github.com\/hyperledger\/fabric\/protos\/msp/\"github.com\/hyperledger\/fabric-sdk-go\/third_party\/github.com\/hyperledger\/fabric\/protos\/msp/g'
    's/\"github.com\/hyperledger\/fabric\/protos\/orderer/\"github.com\/hyperledger\/fabric-sdk-go\/third_party\/github.com\/hyperledger\/fabric\/protos\/orderer/g'
    's/\"github.com\/hyperledger\/fabric\/protos\/ledger/\"github.com\/hyperledger\/fabric-sdk-go\/third_party\/github.com\/hyperledger\/fabric\/protos\/ledger/g'
    's/\"github.com\/hyperledger\/fabric\/protos/\"github.com\/securekey\/fabric-snaps\/internal\/github.com\/hyperledger\/fabric\/protos/g'
    's/\"github.com\/hyperledger\/fabric\//\"github.com\/securekey\/fabric-snaps\/internal\/github.com\/hyperledger\/fabric\//g'
)
eval "INTERNAL_PATH=$THIRDPARTY_INTERNAL_FABRIC_PATH TMP_PROJECT_PATH=$TMP_PROJECT_PATH IMPORT_SUBSTS=\"${PROTOS_INTERNAL_IMPORT_SUBSTS[*]}\" $SCRIPTS_PATH/apply_fabric_protos_internal.sh"

# Cleanup temporary files from patch application
echo "Removing temporary files ..."
rm -Rf $TMP