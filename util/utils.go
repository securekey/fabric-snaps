/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"runtime/debug"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"

	sdkpb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	protos_utils "github.com/hyperledger/fabric/protos/utils"
	"github.com/securekey/fabric-snaps/util/errors"
)

// GetCreatorFromSignedProposal ...
func GetCreatorFromSignedProposal(signedProposal *sdkpb.SignedProposal) ([]byte, error) {

	// check ProposalBytes if nil
	if signedProposal.ProposalBytes == nil {
		return nil, errors.New(errors.GeneralError, "ProposalBytes is nil in SignedProposal")
	}

	proposal, err := protos_utils.GetProposal(signedProposal.ProposalBytes)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "Unmarshal ProposalBytes error")
	}
	// check proposal.Header if nil
	if proposal.Header == nil {
		return nil, errors.New(errors.GeneralError, "Header is nil in Proposal")
	}
	proposalHeader, err := protos_utils.GetHeader(proposal.Header)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Unmarshal HeaderBytes error")
	}
	// check proposalHeader.SignatureHeader if nil
	if proposalHeader.SignatureHeader == nil {
		return nil, errors.New(errors.GeneralError, "signatureHeader is nil in proposalHeader")
	}
	signatureHeader, err := protos_utils.GetSignatureHeader(proposalHeader.SignatureHeader)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Unmarshal SignatureHeader error")
	}

	return signatureHeader.Creator, nil
}

// HandlePanic handles a panic (if any) by populating error response
func HandlePanic(resp *pb.Response, log *logging.Logger, stub shim.ChaincodeStubInterface) {
	if r := recover(); r != nil {

		errObj := errors.Errorf(errors.PanicError, "Check server logs")

		// TODO: Figure out what to log
		log.Errorf("Recovering from panic '%s': %s", errObj.GenerateClientErrorMsg(), string(debug.Stack()))

		errResp := CreateShimResponseFromError(errObj, log, stub)
		resp.Reset()
		resp.Status = errResp.Status
		resp.Message = errResp.Message
	}
}

// CreateShimResponseFromError creates shim response with codedErr as payload
func CreateShimResponseFromError(codedErr errors.Error, log *logging.Logger, stub shim.ChaincodeStubInterface) pb.Response {

	// TODO: We may add logging of all errors at info/warn level here
	return shim.Error(codedErr.GenerateClientErrorMsg())
}
