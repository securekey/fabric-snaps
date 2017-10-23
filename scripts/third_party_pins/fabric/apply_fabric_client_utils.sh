#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# This script pins client and common package families from Hyperledger Fabric into the SDK
# These files are checked into internal paths.
# Note: This script must be adjusted as upstream makes adjustments

IMPORT_SUBSTS=($IMPORT_SUBSTS)

GOIMPORTS_CMD=goimports
GOFILTER_CMD="go run scripts/_go/cmd/gofilter/gofilter.go"

declare -a PKGS=(
    "common/crypto"
    "common/util"
    "common/ledger"

    "sdkpatch/logbridge"

    "core/comm"

    "core/common/ccprovider"

    "core/config"

    "core/chaincode/shim"
    "core/chaincode/shim/ext/encshim"
    "core/chaincode/shim/ext/encshim"
    "core/chaincode/shim/ext/entities"
    "core/chaincode/shim/ext/entities"
    "core/chaincode/shim/ext/entities"
    "core/chaincode/shim/ext/entities"

    "core/ledger/util"
)

declare -a FILES=(
    "common/crypto/random.go"
    "common/crypto/signer.go"

    "common/util/utils.go"

    "common/ledger/ledger_interface.go"
    
    "sdkpatch/logbridge/logbridge.go"

    "core/comm/config.go"
    "core/comm/connection.go"

    "core/config/config.go"
    "core/common/ccprovider/ccprovider.go"

    "core/chaincode/shim/chaincode.go"
    "core/chaincode/shim/handler.go"
    "core/chaincode/shim/inprocstream.go"
    "core/chaincode/shim/interfaces.go"
    "core/chaincode/shim/mockstub.go"
    "core/chaincode/shim/response.go"
    "core/chaincode/shim/ext/encshim/encshim.go"
    "core/chaincode/shim/ext/encshim/interfaces.go"
    "core/chaincode/shim/ext/entities/entities.go"
    "core/chaincode/shim/ext/entities/interfaces.go"
    "core/chaincode/shim/ext/entities/message.go"

    "core/ledger/ledger_interface.go"
    "core/ledger/util/txvalidationflags.go"
)

echo 'Removing current upstream project from working directory ...'
rm -Rf "${INTERNAL_PATH}"
mkdir -p "${INTERNAL_PATH}"

# Create directory structure for packages
for i in "${PKGS[@]}"
do
    mkdir -p $INTERNAL_PATH/${i}
done

# Apply fine-grained patching
gofilter() {
    echo "Filtering: ${FILTER_FILENAME}"
    cp ${TMP_PROJECT_PATH}/${FILTER_FILENAME} ${TMP_PROJECT_PATH}/${FILTER_FILENAME}.bak
    $GOFILTER_CMD -filename "${TMP_PROJECT_PATH}/${FILTER_FILENAME}.bak" \
        -filters "$FILTERS_ENABLED" -fn "$FILTER_FN" -gen "$FILTER_GEN" -type "$FILTER_TYPE" \
        > "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"
} 

echo "Filtering Go sources for allowed functions ..."
FILTERS_ENABLED="fn"

FILTER_FILENAME="common/crypto/random.go"
FILTER_FN="GetRandomNonce,GetRandomBytes"
gofilter

FILTER_FILENAME="common/crypto/signer.go"
FILTER_FN=
gofilter

FILTER_FILENAME="common/util/utils.go"
FILTER_FN="GenerateIDfromTxSHAHash,ComputeSHA256,CreateUtcTimestamp,GenerateUUID,GenerateBytesUUID,idBytesToStr"
gofilter

FILTER_FILENAME="core/ledger/util/txvalidationflags.go"
FILTER_FN="IsValid,IsInvalid,Flag,IsSetTo,NewTxValidationFlags"
gofilter

FILTER_FILENAME="core/chaincode/shim/chaincode.go"
FILTER_FN="userChaincodeStreamGetter,Start,SetupChaincodeLogging,"
FILTER_FN+="StartInProc,getPeerAddress,newPeerClientConnection,chatWithPeer,"
FILTER_FN+="init,GetTxID,GetDecorations,InvokeChaincode,"
FILTER_FN+="GetState,PutState,DelState,handleGetStateByRange,"
FILTER_FN+="GetStateByRange,GetQueryResult,GetHistoryForKey,CreateCompositeKey,"
FILTER_FN+="SplitCompositeKey,createCompositeKey,splitCompositeKey,validateCompositeKeyAttribute,"
FILTER_FN+="validateSimpleKeys,GetStateByPartialCompositeKey,Next,Next,"
FILTER_FN+="HasNext,getResultFromBytes,fetchNextQueryResult,nextResult,"
FILTER_FN+="Close,GetArgs,GetStringArgs,GetFunctionAndParameters,"
FILTER_FN+="GetCreator,GetTransient,GetBinding,GetSignedProposal,"
FILTER_FN+="GetArgsSlice,GetTxTimestamp,SetEvent,SetLoggingLevel,"
FILTER_FN+="LogLevel,NewLogger,SetLevel,IsEnabledFor,"
FILTER_FN+="Debug,Info,Notice,Warning,"
FILTER_FN+="Error,Critical,Debugf,Infof,"
FILTER_FN+="Noticef,Warningf,Errorf,Criticalf"
gofilter
sed -i'' -e 's/var ccInfoCache = NewCCInfoCache(ccInfoFSProvider)//g' "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"
sed -i '/pb "github.com\// a pbinternal "github.com\/securekey\/fabric-snaps\/internal\/github.com\/hyperledger\/fabric\/protos\/peer"' "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"
sed -i'' -e 's/pb.NewChaincodeSupportClient/pbinternal.NewChaincodeSupportClient/g' "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"
sed -i'' -e 's/pb.ChaincodeMessage/pbinternal.ChaincodeMessage/g' "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"
sed -i'' -e 's/pb.QueryResponse/pbinternal.QueryResponse/g' "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"
sed -i'' -e 's/pb.QueryResultBytes/pbinternal.QueryResultBytes/g' "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"

FILTER_FILENAME="core/chaincode/shim/handler.go"
sed -i '/pb "github.com\// a pbinternal "github.com\/securekey\/fabric-snaps\/internal\/github.com\/hyperledger\/fabric\/protos\/peer"' "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"
sed -i'' -e 's/pb.ChaincodeMessage/pbinternal.ChaincodeMessage/g' "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"
sed -i'' -e 's/pb.GetStateByRange/pbinternal.GetStateByRange/g' "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"
sed -i'' -e 's/pb.QueryStateClose/pbinternal.QueryStateClose/g' "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"
sed -i'' -e 's/pb.QueryResponse/pbinternal.QueryResponse/g' "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"
sed -i'' -e 's/pb.QueryStateNext/pbinternal.QueryStateNext/g' "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"
sed -i'' -e 's/pb.GetQueryResult/pbinternal.GetQueryResult/g' "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"
sed -i'' -e 's/pb.GetHistoryForKey/pbinternal.GetHistoryForKey/g' "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"
sed -i'' -e 's/pb.PutStateInfo/pbinternal.PutStateInfo/g' "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"

FILTER_FILENAME="core/chaincode/shim/inprocstream.go"
sed -i '/pb "github.com\// a pbinternal "github.com\/securekey\/fabric-snaps\/internal\/github.com\/hyperledger\/fabric\/protos\/peer"' "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"
sed -i'' -e 's/pb.ChaincodeMessage/pbinternal.ChaincodeMessage/g' "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"

FILTER_FILENAME="core/common/ccprovider/ccprovider.go"
FILTER_FN=Reset,String,ProtoMessage
gofilter
sed -i'' -e 's/var ccInfoCache = NewCCInfoCache(ccInfoFSProvider)//g' "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"

FILTER_FILENAME="core/comm/connection.go"
FILTER_FN=NewClientConnectionWithAddress,InitTLSForShim,MaxRecvMsgSize
gofilter


FILTER_FILENAME="core/ledger/ledger_interface.go"
sed -i'' -e 's/"github.com\/hyperledger\/fabric\/protos\/ledger\/rwset"/ "github.com\/hyperledger\/fabric-sdk-go\/third_party\/github.com\/hyperledger\/fabric\/protos\/ledger\/rwset" /g' "${TMP_PROJECT_PATH}/${FILTER_FILENAME}"

echo "Filtering Go sources for allowed declarations ..."
FILTERS_ENABLED="gen,type"
FILTER_TYPE="IMPORT,CONST"
# Allow no declarations
FILTER_GEN=


# Apply patching
echo "Patching import paths on upstream project ..."
WORKING_DIR=$TMP_PROJECT_PATH FILES="${FILES[@]}" IMPORT_SUBSTS="${IMPORT_SUBSTS[@]}" scripts/third_party_pins/common/apply_import_patching.sh

echo "Inserting modification notice ..."
WORKING_DIR=$TMP_PROJECT_PATH FILES="${FILES[@]}" scripts/third_party_pins/common/apply_header_notice.sh

# Copy patched project into internal paths
echo "Copying patched upstream project into working directory ..."
for i in "${FILES[@]}"
do
    TARGET_PATH=`dirname $INTERNAL_PATH/${i}`
    cp $TMP_PROJECT_PATH/${i} $TARGET_PATH
done
