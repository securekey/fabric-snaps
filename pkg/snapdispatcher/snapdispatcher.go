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
	"github.com/securekey/fabric-snaps/api/config"
	snapInterfaces "github.com/securekey/fabric-snaps/api/interfaces"
	snapProtos "github.com/securekey/fabric-snaps/api/protos"
	"github.com/securekey/fabric-snaps/pkg/snapdispatcher/registry"
	grpc "google.golang.org/grpc"
	cred "google.golang.org/grpc/credentials"
)

var logger = logging.MustGetLogger("snap-server")

type snapServer struct {
	registry registry.SnapsRegistry
}

//SnapServer Invoke...
func (ss *snapServer) Invoke(ctx context.Context, ireq *snapProtos.Request) (*snapProtos.Response, error) {
	//Snap name - required
	snapName := ireq.SnapName
	logger.Debugf("Invoking snap %s ", snapName)
	handler, err := ss.getRegisteredSnapHandler(snapName)
	if err != nil {
		return nil, err
	}

	//Create snap stub and pass it in
	snapStub := snapInterfaces.NewSnapStub(ireq.Args)
	//invoke snap
	invokeResponse := handler.Invoke(snapStub)
	//response from invoke
	irPayload := [][]byte{invokeResponse.Payload}
	ir := snapProtos.Response{Payload: irPayload, Status: snapProtos.Status(invokeResponse.Status)}
	return &ir, nil
}

//getRegisteredSnapHandler retrurns registration status and invoke interface
func (ss *snapServer) getRegisteredSnapHandler(snapName string) (shim.Chaincode, error) {
	registeredSnap := ss.registry.GetSnap(snapName)
	if registeredSnap == nil {
		return nil, fmt.Errorf("Snap [%s] not found", snapName)
	}
	if !registeredSnap.Enabled {
		return nil, fmt.Errorf("Snap [%s] is disabled", snapName)
	}
	return registeredSnap.Snap, nil
}

//startSnapServer ... grpc
func startSnapServer(registry registry.SnapsRegistry) error {
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
	snapProtos.RegisterSnapServer(s, &snapServer{registry: registry})
	if config.IsTLSEnabled() {
		logger.Infof("Start Snap Server grpc with tls on port:%s\n", config.GetSnapServerPort())
	} else {
		logger.Infof("Start Snap Server on port:%s\n", config.GetSnapServerPort())
	}
	go s.Serve(lis)
	return nil

}
