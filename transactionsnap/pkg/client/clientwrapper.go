/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazyref"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/util/errors"
)

const (
	backoff = 250 * time.Millisecond
)

// clientWrapper is an implementation of the api.Client interface that, for each function, retrieves the latest
// available client from the cache and adds a reference to the client to prevent the client from being closed while
// the operation is in progress. Once the invocation has completed, the reference is released.
type clientWrapper struct {
	channelID string
}

func (c *clientWrapper) EndorseTransaction(endorseRequest *api.EndorseTxRequest) (*channel.Response, errors.Error) {
	client, err := c.get()
	if err != nil {
		return nil, err
	}
	defer client.Release()

	return client.endorseTransaction(endorseRequest)
}

func (c *clientWrapper) CommitTransaction(endorseRequest *api.EndorseTxRequest, registerTxEvent bool, callback api.EndorsedCallback) (*channel.Response, bool, errors.Error) {
	client, err := c.get()
	if err != nil {
		return nil, false, err
	}
	defer client.Release()

	return client.commitTransaction(endorseRequest, registerTxEvent, callback)
}

func (c *clientWrapper) VerifyTxnProposalSignature(s []byte) errors.Error {
	client, err := c.get()
	if err != nil {
		return err
	}
	defer client.Release()

	return client.verifyTxnProposalSignature(s)
}

func (c *clientWrapper) GetLocalPeer() (fabApi.Peer, error) {
	client, err := c.get()
	if err != nil {
		return nil, err
	}
	defer client.Release()

	return client.getLocalPeer()
}

func (c *clientWrapper) ChannelConfig() (fabApi.ChannelCfg, error) {
	client, err := c.get()
	if err != nil {
		return nil, err
	}
	defer client.Release()

	return client.channelConfig()
}

func (c *clientWrapper) EventService() (fabApi.EventService, error) {
	client, err := c.get()
	if err != nil {
		return nil, err
	}
	defer client.Release()

	return client.eventService()
}

func (c *clientWrapper) GetDiscoveredPeer(url string) (fabApi.Peer, error) {
	client, err := c.get()
	if err != nil {
		return nil, err
	}
	defer client.Release()

	return client.getDiscoveredPeer(url)
}

func (c *clientWrapper) get() (*clientImpl, errors.Error) {
	for {
		client, err := c.getClient()
		if err != nil {
			return nil, err
		}

		if !client.Acquire() {
			logger.Infof("Could not acquire a reference to client [%s] on channel [%s]. Trying again in %s...", client.configHash, c.channelID, backoff)
			time.Sleep(backoff)
		} else {
			return client, nil
		}
	}
}

func (c *clientWrapper) getClient() (*clientImpl, errors.Error) {
	ref, err := cache.Get(newCacheKey(c.channelID, CfgProvider, ServiceProviderFactory))
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Got error while getting item from cache")
	}

	clientRef := ref.(*lazyref.Reference)
	client, err := clientRef.Get()
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "error getting client")
	}
	return client.(*clientImpl), nil
}
