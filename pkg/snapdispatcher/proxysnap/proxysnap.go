/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package proxysnap

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	logging "github.com/op/go-logging"
	client "github.com/securekey/fabric-snaps/api/client"
	"github.com/securekey/fabric-snaps/api/protos"
)

var logger = logging.MustGetLogger("proxy-snap")

type snapsClientProvider func(url string) client.SnapsClient

// proxySnap invokes a remote snap
type proxySnap struct {
	name      string
	url       string
	newClient snapsClientProvider
}

// NewSnap - create new instance of ProxySnap
func NewSnap(tlsEnabled bool, tlsRootCert string) shim.Chaincode {
	return newSnap(func(url string) client.SnapsClient {
		return client.NewSnapsClient(url, tlsEnabled, tlsRootCert, "")
	})
}

func newSnap(client snapsClientProvider) shim.Chaincode {
	return &proxySnap{newClient: client}
}

// Init initializes the snap
// arg[0] - Snap name
// arg[1] - Remote snap URL
func (s *proxySnap) Init(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	if len(args) < 2 {
		return shim.Error("expecting snap name and remote snap URL as arguments")
	}

	s.name = string(args[0])
	s.url = string(args[1])

	logger.Infof("Remote Snap [%s] Initialized - URL: %s", s.name, s.url)

	return shim.Success(nil)
}

// Invoke the snap. The configured remote snap is invoked. All args are passed along.
func (s *proxySnap) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Debugf("Invoking remote snap [%s] at [%s]\n", s.name, s.url)

	snapsClient := s.newClient(s.url)
	defer snapsClient.Disconnect()

	response := snapsClient.Send(&protos.Request{SnapName: s.name, Args: stub.GetArgs()})
	if response.Status != shim.OK {
		return shim.Error(response.Error)
	}

	return shim.Success(response.Payload[0])
}
