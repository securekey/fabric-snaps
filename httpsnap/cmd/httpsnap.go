/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"strings"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"

	shim "github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	httpsnapservice "github.com/securekey/fabric-snaps/httpsnap/cmd/httpsnapservice"
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
	logger.Info("Http snap loaded.")
	return shim.Success(nil)
}

// Invoke should be called with 4 mandatory arguments (and 2 optional ones):
// args[0] - Function (currently not used)
// args[1] - URL
// args[2] - Content-Type
// args[3] - Request Body
// args[4] - Named Client (optional)
// args[5] - Pin set (optional)
func (httpsnap *HTTPSnap) Invoke(stub shim.ChaincodeStubInterface) pb.Response {

	httpservice, err := httpsnapservice.Get(stub.GetChannelID())
	if err != nil {
		return shim.Error(err.Error())
	}

	_, args := stub.GetFunctionAndParameters()

	if len(args) < 3 {
		return shim.Error("Missing URL parameter, content type and/or request body")
	}

	requestURL := args[0]
	if requestURL == "" {
		return shim.Error("Missing URL parameter")
	}

	contentType := args[1]
	if contentType == "" {
		return shim.Error("Missing content type")
	}

	requestBody := args[2]
	if requestBody == "" {
		return shim.Error("Missing request body")
	}

	// Optional parameter: named client (used for determining parameters for TLS configuration)
	client := ""
	if len(args) >= 4 {
		client = string(args[3])
	}

	// Optional parameter: pin set(comma separated)
	pins := []string{}
	if len(args) >= 5 && args[4] != "" && strings.TrimSpace(args[4]) != "" {
		pins = strings.Split(args[4], ",")
	}

	response, err := httpservice.Invoke(httpsnapservice.HTTPServiceInvokeRequest{RequestURL: requestURL, ContentType: contentType,
		RequestBody: requestBody, NamedClient: client, PinSet: pins})

	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(response)

}

func main() {
}
