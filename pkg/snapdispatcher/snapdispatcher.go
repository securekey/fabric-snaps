/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package snapdispatcher

import (
	"fmt"

	context "golang.org/x/net/context"

	"net"
	"strings"

	shim "github.com/hyperledger/fabric/core/chaincode/shim"
	logging "github.com/op/go-logging"
	snap_interfaces "github.com/securekey/fabric-snaps/api/interfaces"
	snap_protos "github.com/securekey/fabric-snaps/api/protos"
	config "github.com/securekey/fabric-snaps/cmd/config"
	grpc "google.golang.org/grpc"
	cred "google.golang.org/grpc/credentials"
)

var logger = logging.MustGetLogger("snap-server")

type snapServer struct {
}

//SnapServer Invoke...

func (ss *snapServer) Invoke(ctx context.Context, ireq *snap_protos.Request) (*snap_protos.Response, error) {
	//Snap name - required
	snapName := ireq.SnapName
	logger.Debugf("Invoking snap %s ", snapName)
	isRegistered, handler := getRegisteredSnapHandler(snapName)
	if isRegistered == false {
		return nil, fmt.Errorf("Snap %s was not registered", snapName)
	}
	//Snap receiver interface is requred
	if handler == nil {
		return nil, fmt.Errorf("Handler (Snap interface) was not configured for %s", snapName)
	}

	//Create snap stub and pass it in
	snapStub := snap_interfaces.NewSnapStub(ireq.Args)
	//invoke snap
	invokeResponse := handler.Invoke(snapStub)
	//response from invoke
	irPayload := [][]byte{invokeResponse.Payload}
	ir := snap_protos.Response{Payload: irPayload}
	return &ir, nil
}

//getRegisteredSnapHandler retrurns registration status and invoke interface
func getRegisteredSnapHandler(snapName string) (bool, shim.Chaincode) {
	registeredSnap := config.GetSnapConfig(snapName)
	if registeredSnap != nil {
		return true, registeredSnap.Snap
	}
	return false, nil
}

//StartSnapServer ... grpc
func StartSnapServer() error {
	if strings.TrimSpace(config.GetSnapServerPort()) == "" {
		logger.Error("GRPC port was not set for snap invoke server")
		return fmt.Errorf("GRPC port was not set for snap invoke server")
	}
	lis, err := net.Listen("tcp", ":"+config.GetSnapServerPort())
	if err != nil {
		return fmt.Errorf("Snap Server error failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	if config.IsTLSEnabled() {
		creds, err := cred.NewServerTLSFromFile(config.GetTLSCertPath(), config.GetTLSKeyPath())
		if err != nil {
			return fmt.Errorf("Snap Server error failed to generate Tls credentials %v", err)
		}
		logger.Info("Snap Server TLS credentials successfully loaded")
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}
	s := grpc.NewServer(opts...)
	snap_protos.RegisterSnapServer(s, &snapServer{})
	if config.IsTLSEnabled() {
		logger.Infof("Start Snap Server grpc with tls on port:%s\n", config.GetSnapServerPort())
	} else {
		logger.Infof("Start Snap Server on port:%s\n", config.GetSnapServerPort())
	}
	go s.Serve(lis)
	return nil

}
