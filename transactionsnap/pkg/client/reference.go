/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	apisdk "github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/api"
	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazyref"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"time"
)

// Ref config reference
type Ref struct {
	*lazyref.Reference
	clientConfig           *clientImpl
	channelID              string
	txnSnapConfig          api.Config
	serviceProviderFactory apisdk.ServiceProviderFactory
}

// NewRef returns a new membership reference
func NewRef(refresh time.Duration, channelID string, txnSnapConfig api.Config,
	serviceProviderFactory apisdk.ServiceProviderFactory) *Ref {
	clnt := &clientImpl{txnSnapConfig: txnSnapConfig}
	clnt.initialize(channelID, serviceProviderFactory)
	ref := &Ref{
		channelID:              channelID,
		txnSnapConfig:          txnSnapConfig,
		serviceProviderFactory: serviceProviderFactory,
		clientConfig:           clnt,
	}

	ref.Reference = lazyref.New(
		ref.initializer(),
		lazyref.WithRefreshInterval(lazyref.InitImmediately, refresh),
	)

	return ref
}

func (ref *Ref) initializer() lazyref.Initializer {
	return func() (interface{}, error) {
		err := ref.clientConfig.initialize(ref.channelID, ref.serviceProviderFactory)
		if err != nil {
			return nil, err
		}
		return ref.clientConfig, nil
	}
}
