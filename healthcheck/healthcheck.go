/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package healthcheck

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

var logger = shim.NewLogger("healthcheck")

const (
	// FMPScc HealthCheck
	FMPScc = "FMPScc"
	// ConfigurationScc Healthcheck
	ConfigurationScc = "ConfigurationScc"
	// TxDelegationScc Healthcheck
	TxDelegationScc = "TxDelegationScc"
)

// SmokeTestResult is a structure representing the results of a SmokeTest
type SmokeTestResult struct {
	Message string `json:"message,omitempty"`
	Status  int    `json:"status,omitempty"`
}

// SmokeTest is a health check function that returns the status of Snap it is called up
func SmokeTest(extScc string, stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	switch extScc {
	case FMPScc:
		logger.Info("Executing FMP SCC smoke test...")
		return healthcheckFmpScc()
	case ConfigurationScc:
		logger.Info("Executing Confirguration SCC smoke test...")
		return healthcheckConfigurationScc()
	case TxDelegationScc:
		logger.Info("Executing Tx Delegation SCC smoke test...")
		return healthcheckTxDelegationScc()
	default:
		logger.Info("Smoke test of unrecognized ExtSCC '%s' ...")
		defaultResult := &SmokeTestResult{
			fmt.Sprintf("%s Healthcheck had nothing to run. Returning empty success response..", extScc),
			shim.OK,
		}
		payload, err := json.Marshal(defaultResult)
		if err != nil {
			return shim.Error(fmt.Sprintf("Error occurred while Marshalling: %s; stub=%s ; args=%s", err, stub, args))
		}
		return shim.Success(payload)
	}

}

// UnmarshalEchoResponse will JSON Unmarshal an object of type []byte
func UnmarshalEchoResponse(objBytes []byte) (*SmokeTestResult, error) {
	obj := &SmokeTestResult{}
	err := json.Unmarshal(objBytes, obj)
	return obj, err
}

// Healthcheck FMPScc
func healthcheckFmpScc(
//stub shim.ChaincodeStubInterface, args [][]byte
) pb.Response {
	//todo add FMPScc healthcheck logic here
	return shim.Success(nil)
}

// Healthcheck ConfigurationScc
func healthcheckConfigurationScc(
//stub shim.ChaincodeStubInterface, args [][]byte
) pb.Response {
	//todo add ConfigurationScc healthcheck logic here
	return shim.Success(nil)
}

// Healthcheck TxDelegationScc
func healthcheckTxDelegationScc(
//stub shim.ChaincodeStubInterface, args [][]byte
) pb.Response {
	//todo add TxDelegationScc healthcheck logic here
	return shim.Success(nil)
}
