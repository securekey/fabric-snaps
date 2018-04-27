/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package httpsnapservice

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strconv"

	commtls "github.com/hyperledger/fabric-sdk-go/pkg/core/config/comm/tls"
	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazycache"
)

// CertPoolCacheKey channel config reference cache key
type CertPoolCacheKey interface {
	lazycache.Key
	UseSystemPool() bool
}

// certPoolCacheKey holds a key for the cert pool cache
type certPoolCacheKey struct {
	key           string
	useSystemPool bool
}

// NewCertPoolCache a cache of cert pool instances
// It is expected to only have two values: one with system cert pool
// enabled and another with it disabled
// This allows dynamic configuration to toggle this setting
func NewCertPoolCache() *lazycache.Cache {
	initializer := func(key lazycache.Key) (interface{}, error) {
		ck, ok := key.(CertPoolCacheKey)
		if !ok {
			return nil, errors.New("unexpected cache key")
		}

		logger.Infof("Initializing cert pool cache")

		return commtls.NewCertPool(ck.UseSystemPool()), nil
	}

	return lazycache.New("Certpool_Cache", initializer)
}

// NewCertPoolCacheKey returns a new CacheKey
func NewCertPoolCacheKey(useSystemPool bool) CertPoolCacheKey {
	return &certPoolCacheKey{
		key:           strconv.FormatBool(useSystemPool),
		useSystemPool: useSystemPool,
	}
}

// String returns the key as a string
func (k *certPoolCacheKey) String() string {
	return k.key
}

// UseSystemPool returns whether to use the system cert pool
func (k *certPoolCacheKey) UseSystemPool() bool {
	return k.useSystemPool
}

func decodeCerts(pemCertsList []string) []*x509.Certificate {
	var certs []*x509.Certificate
	for _, pemCertsString := range pemCertsList {
		pemCerts := []byte(pemCertsString)
		for len(pemCerts) > 0 {
			var block *pem.Block
			block, pemCerts = pem.Decode(pemCerts)
			if block == nil {
				break
			}
			if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
				continue
			}

			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				continue
			}

			certs = append(certs, cert)
		}
	}

	return certs
}
