/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"crypto/x509"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/securekey/fabric-snaps/util/errors"
	"github.com/spf13/viper"
)

// Config configuration interface
type Config interface {
	GetLocalPeer() (*PeerConfig, errors.Error)
	GetMspID() string
	GetMspConfigPath() string
	GetTLSRootCertPath() string
	GetTLSRootCert() *x509.Certificate
	GetTLSCertPath() string
	GetTLSCert() *x509.Certificate
	GetTLSCertPem() []byte
	GetTLSKeyPath() string
	GetConfigPath(path string) string
	GetPeerConfig() *viper.Viper
	GetConfigBytes() []byte
	GetCryptoProvider() (string, errors.Error)
	GetEndorserSelectionMaxAttempts() int
	GetEndorserSelectionInterval() time.Duration
	RetryOpts() retry.Opts
	CCErrorRetryableCodes() ([]int32, errors.Error)
	GetClientCacheRefreshInterval() time.Duration
}

// PeerConfig represents the server addresses of a fabric peer
type PeerConfig struct {
	Host  string
	Port  int
	MSPid []byte
}
