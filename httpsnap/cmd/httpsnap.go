/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"encoding/json"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/common/logging"

	shim "github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/httpsnap/api"
	httpsnapservice "github.com/securekey/fabric-snaps/httpsnap/cmd/httpsnapservice"
	"github.com/securekey/fabric-snaps/util"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("httpsnap")

//HTTPSnap implementation
type HTTPSnap struct {
}

// New chaincode implementation
func New() shim.Chaincode {
	return &HTTPSnap{}
}

// Init snap
func (httpsnap *HTTPSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {

	logger.Info("Snap configuration loaded.")
	return shim.Success(nil)
}

// Invoke should be called with 4 mandatory arguments (and 2 optional ones):
// args[0] - Function (currently not used)
// args[1] - HttpSnapRequest
func (httpsnap *HTTPSnap) Invoke(stub shim.ChaincodeStubInterface) (resp pb.Response) {

	defer util.HandlePanic(&resp, logger, stub)

	httpservice, err := httpsnapservice.Get(stub.GetChannelID())
	if err != nil {
		return util.CreateShimResponseFromError(errors.WithMessage(errors.SystemError, err, "Failed to get http snap service"), logger, stub)
	}

	args := stub.GetArgs()

	//first arg is function name; the second one is HttpSnapRequest
	if len(args) < 2 {
		return util.CreateShimResponseFromError(errors.New(errors.MissingRequiredParameterError, "Missing function name and/or http snap request"), logger, stub)
	}

	if args[1] == nil || len(args[1]) == 0 {
		return util.CreateShimResponseFromError(errors.New(errors.MissingRequiredParameterError, "Http Snap Request is nil or empty"), logger, stub)
	}

	request, codedErr := getHTTPSnapRequest(args[1])
	if codedErr != nil {
		return util.CreateShimResponseFromError(codedErr, logger, stub)
	}

	response, codedErr := httpservice.Invoke(httpsnapservice.HTTPServiceInvokeRequest{RequestURL: request.URL, RequestHeaders: request.Headers,
		RequestBody: request.Body, NamedClient: request.NamedClient, PinSet: request.PinSet})

	if codedErr != nil {
		return util.CreateShimResponseFromError(codedErr, logger, stub)
	}

	resp = shim.Success(response)

	return
}

// helper method for unmarshalling http snap request
func getHTTPSnapRequest(reqBytes []byte) (*api.HTTPSnapRequest, errors.Error) {
	var req api.HTTPSnapRequest
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, errors.Wrap(errors.UnmarshallError, err, "Failed to unmarshal http snap request")
	}
	return &req, nil
}

func main() {
}
