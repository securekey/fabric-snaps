/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"time"

	"github.com/spf13/viper"
)

// Config configuration interface
type Config interface {
	GetLocalPeer() (*PeerConfig, error)
	GetMspID() string
	GetTLSRootCertPath() string
	GetTLSCertPath() string
	GetTLSKeyPath() string
	GetMembershipPollInterval() time.Duration
	GetGRPCProtocol() string
	GetConfigPath(path string) string
	GetPeerConfig() *viper.Viper
}

// PeerConfig represents the server addresses of a fabric peer
type PeerConfig struct {
	Host      string
	Port      int
	EventHost string
	EventPort int
	MSPid     []byte
}
