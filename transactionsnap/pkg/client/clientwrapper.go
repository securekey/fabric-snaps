/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"regexp"
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

var retryableErrors = []*regexp.Regexp{
	regexp.MustCompile("(?i).*sign proposal failed.*"),
	regexp.MustCompile("(?i).*sign failed.*"),
	regexp.MustCompile("(?i).*Private key not found.*"),
	regexp.MustCompile("(?i).*Key not found.*"),
}

// clientWrapper is an implementation of the api.Client interface that, for each function, retrieves the latest
// available client from the cache and adds a reference to the client to prevent the client from being closed while
// the operation is in progress. Once the invocation has completed, the reference is released.
type clientWrapper struct {
	channelID string
}

func (c *clientWrapper) EndorseTransaction(endorseRequest *api.EndorseTxRequest) (*channel.Response, errors.Error) {
	endorseTransaction := func(endorseRequest *api.EndorseTxRequest) (*channel.Response, errors.Error) {
		client, err := c.get()
		if err != nil {
			return nil, err
		}
		defer client.Release()

		return client.endorseTransaction(endorseRequest)
	}

	resp, err := endorseTransaction(endorseRequest)
	if isRetryable(err) {
		c.clearCache()
		resp, err = endorseTransaction(endorseRequest)
	}
	return resp, err
}

func (c *clientWrapper) CommitTransaction(endorseRequest *api.EndorseTxRequest, registerTxEvent bool, callback api.EndorsedCallback) (*channel.Response, bool, errors.Error) {

	commitTx := func(endorseRequest *api.EndorseTxRequest, registerTxEvent bool, callback api.EndorsedCallback) (*channel.Response, bool, errors.Error) {
		client, err := c.get()
		if err != nil {
			return nil, false, err
		}
		defer client.Release()

		return client.commitTransaction(endorseRequest, registerTxEvent, callback)
	}

	resp, commit, err := commitTx(endorseRequest, registerTxEvent, callback)
	if isRetryable(err) {
		c.clearCache()
		resp, commit, err = commitTx(endorseRequest, registerTxEvent, callback)
	}
	return resp, commit, err
}

func (c *clientWrapper) VerifyTxnProposalSignature(s []byte) errors.Error {
	verifySignature := func(s []byte) errors.Error {
		client, err := c.get()
		if err != nil {
			return err
		}
		defer client.Release()

		return client.verifyTxnProposalSignature(s)
	}

	err := verifySignature(s)
	if isRetryable(err) {
		c.clearCache()
		err = verifySignature(s)
	}
	return err
}

func (c *clientWrapper) GetLocalPeer() (fabApi.Peer, error) {
	getLocalPeer := func() (fabApi.Peer, error) {
		client, err := c.get()
		if err != nil {
			return nil, err
		}
		defer client.Release()

		return client.getLocalPeer()
	}

	peer, err := getLocalPeer()
	if isRetryable(err) {
		c.clearCache()
		peer, err = getLocalPeer()
	}
	return peer, err
}

func (c *clientWrapper) ChannelConfig() (fabApi.ChannelCfg, error) {
	channelCfg := func() (fabApi.ChannelCfg, error) {
		client, err := c.get()
		if err != nil {
			return nil, err
		}
		defer client.Release()

		return client.channelConfig()
	}

	cfg, err := channelCfg()
	if isRetryable(err) {
		c.clearCache()
		cfg, err = channelCfg()
	}
	return cfg, err
}

func (c *clientWrapper) EventService() (fabApi.EventService, error) {
	eventService := func() (fabApi.EventService, error) {
		client, err := c.get()
		if err != nil {
			return nil, err
		}
		defer client.Release()

		return client.eventService()
	}

	svc, err := eventService()
	if isRetryable(err) {
		c.clearCache()
		svc, err = eventService()
	}
	return svc, err
}

func (c *clientWrapper) GetDiscoveredPeer(url string) (fabApi.Peer, error) {
	getDiscoveredPeer := func(url string) (fabApi.Peer, error) {
		client, err := c.get()
		if err != nil {
			return nil, err
		}
		defer client.Release()

		return client.getDiscoveredPeer(url)
	}

	peer, err := getDiscoveredPeer(url)
	if isRetryable(err) {
		c.clearCache()
		peer, err = getDiscoveredPeer(url)
	}
	return peer, err
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

func (c *clientWrapper) clearCache() {
	cache.Delete(newCacheKey(c.channelID, CfgProvider, ServiceProviderFactory))
}

//isRetryable matches error message predefined set of error patterns
func isRetryable(e interface{}) bool {

	if e == nil {
		return false
	}

	err, ok := e.(errors.Error)
	if ok {
		return matchRetryableErrors(err.GenerateClientErrorMsg())
	}

	er, ok := e.(error)
	if ok {
		return matchRetryableErrors(er.Error())
	}

	return false
}

func matchRetryableErrors(msg string) bool {
	for _, v := range retryableErrors {
		if v.MatchString(msg) {
			return true
		}
	}
	return false
}
