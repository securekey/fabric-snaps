/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package proxysnap

import (
	"context"
	"errors"
	"fmt"

	"sync"

	"github.com/securekey/fabric-snaps/api/protos"
	"google.golang.org/grpc"
	creds "google.golang.org/grpc/credentials"
)

// SnapsClient sends a request to a remote Snaps container
type SnapsClient interface {
	// Send sends the request to the Snap
	Send(request *protos.Request) protos.Response

	// Disconnect closes the connection
	Disconnect()
}

type snapsClient struct {
	url                string
	tlsEnabled         bool
	tlsCertFile        string
	serverHostOverride string
	connection         *grpc.ClientConn
	mutex              sync.RWMutex
}

// NewSnapsClient creates a new Snaps client
func NewSnapsClient(url string, tlsEnabled bool, tlsCertFile string, serverHostOverride string) SnapsClient {
	return &snapsClient{
		url:                url,
		tlsEnabled:         tlsEnabled,
		serverHostOverride: serverHostOverride,
		tlsCertFile:        tlsCertFile,
	}
}

func (c *snapsClient) Send(request *protos.Request) protos.Response {
	conn, err := c.connect()
	if err != nil {
		errMsg := fmt.Sprintf("Failed to connect to snaps dispatcher at %s. Error: %v", c.url, err)
		logger.Errorf(errMsg)
		return protos.Response{Error: err.Error()}
	}

	// Invoke snap using snaps client
	client := protos.NewSnapClient(conn)
	response, err := client.Invoke(context.Background(), request)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to invoke snaps dispatcher. Error: %v", err)
		logger.Warning(errMsg)
		return protos.Response{Error: err.Error()}
	}

	return protos.Response{Status: response.Status, Payload: response.Payload}
}

func (c *snapsClient) Disconnect() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.connection != nil {
		c.connection.Close()
		c.connection = nil
	}
}

// Get connection to snaps dispatcher. If the client is already connected
// then the existing connection is returned, otherwise a new connection is created.
func (c *snapsClient) connect() (*grpc.ClientConn, error) {
	var connection *grpc.ClientConn

	c.mutex.RLock()
	connection = c.connection
	c.mutex.RUnlock()

	if connection != nil {
		return connection, nil
	}

	// read snaps dispatcher port
	if c.url == "" {
		logger.Warningf("snapsDispatcherAddress was not set. Set property: 'chaincode.system.config.snapsscc.snapsDispatcherAddress'")
		return nil, errors.New("Error detecting snaps dispatcher address from property chaincode.system.config.snapsscc.snapsDispatcherAddress")
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// grpc connection options
	var opts []grpc.DialOption
	if c.tlsEnabled {
		creds, err := creds.NewClientTLSFromFile(c.tlsCertFile, c.serverHostOverride)
		if err != nil {
			return nil, fmt.Errorf("Failed to create snaps dispatcher tls client from file: %s Error: %v", c.tlsCertFile, err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		// TLS disabled
		opts = append(opts, grpc.WithInsecure())
	}

	logger.Infof("Dialing snaps dispatcher on: %s", c.url)

	connection, err := grpc.Dial(c.url, opts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to dial snaps dispatcher at %s. Error: %v", c.url, err)
	}

	c.connection = connection

	return connection, err
}
