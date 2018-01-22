/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"encoding/json"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"

	shim "github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/httpsnap/api"
	httpsnapservice "github.com/securekey/fabric-snaps/httpsnap/cmd/httpsnapservice"
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
func (httpsnap *HTTPSnap) Invoke(stub shim.ChaincodeStubInterface) pb.Response {

	httpservice, err := httpsnapservice.Get(stub.GetChannelID())
	if err != nil {
		return shim.Error(err.Error())
	}

	args := stub.GetArgs()

	//first arg is function name; the second one is HttpSnapRequest
	if len(args) < 2 {
		return shim.Error("Missing function name and/or http snap request")
	}

	if args[1] == nil || len(args[1]) == 0 {
		return shim.Error("Http Snap Request is nil or empty")
	}

	request, err := getHTTPSnapRequest(args[1])
	if err != nil {
		return shim.Error(err.Error())
	}

	if request.URL == "" {
		return shim.Error("Missing URL parameter")
	}

	if len(request.Headers) == 0 {
		return shim.Error("Missing request headers")
	}

	if _, ok := request.Headers["Content-Type"]; !ok {
		return shim.Error("Missing required Content-Type header")
	}

	if val, ok := request.Headers["Content-Type"]; ok && val == "" {
		return shim.Error("Content-Type header is empty")
	}

	if request.Body == "" {
		return shim.Error("Missing request body")
	}

	response, err := httpservice.Invoke(httpsnapservice.HTTPServiceInvokeRequest{RequestURL: request.URL, RequestHeaders: request.Headers,
		RequestBody: request.Body, NamedClient: request.NamedClient, PinSet: request.PinSet})

	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(response)

}

// helper method for unmarshalling http snap request
func getHTTPSnapRequest(reqBytes []byte) (*api.HTTPSnapRequest, error) {
	var req api.HTTPSnapRequest
	err := json.Unmarshal(reqBytes, &req)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "Failed json.Unmarshal")
	}
	return &req, nil
}

func main() {
}
