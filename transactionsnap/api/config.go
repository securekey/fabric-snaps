/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"crypto/x509"
	"time"

	"github.com/spf13/viper"
)

// Config configuration interface
type Config interface {
	GetLocalPeer() (*PeerConfig, error)
	GetMspID() string
	GetMspConfigPath() string
	GetTLSRootCertPath() string
	GetTLSRootCert() *x509.Certificate
	GetTLSCertPath() string
	GetTLSCert() *x509.Certificate
	GetTLSKeyPath() string
	GetGRPCProtocol() string
	GetConfigPath(path string) string
	GetPeerConfig() *viper.Viper
	GetConfigBytes() []byte
	GetEndorserSelectionMaxAttempts() int
	GetEndorserSelectionInterval() time.Duration
	GetEndorsementMaxAttempts() int
	GetEndorsementRetryInterval() time.Duration
	GetCommitTimeout() time.Duration
}

// PeerConfig represents the server addresses of a fabric peer
type PeerConfig struct {
	Host      string
	Port      int
	EventHost string
	EventPort int
	MSPid     []byte
}
