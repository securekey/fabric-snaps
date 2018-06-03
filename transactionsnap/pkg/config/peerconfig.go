/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config/endpoint"
	transactionsnapApi "github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/util/errors"
)

// PeerConfigs represents a list of peers. It implements the sort interface
type PeerConfigs []transactionsnapApi.PeerConfig

func (p PeerConfigs) Len() int {
	return len(p)
}

func (p PeerConfigs) Less(i, j int) bool {
	return p[i].Host < p[j].Host
}

func (p PeerConfigs) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

// NewNetworkPeer creates a NetworkPeer
func NewNetworkPeer(peerConfig *fabApi.PeerConfig, mspID string, pem []byte) (*fabApi.NetworkPeer, error) {
	networkPeer := &fabApi.NetworkPeer{PeerConfig: *peerConfig, MSPID: mspID}
	networkPeer.TLSCACerts = endpoint.TLSConfig{Pem: string(pem)}
	if err := networkPeer.TLSCACerts.LoadBytes(); err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "error loading TLSCACert bytes")
	}
	return networkPeer, nil
}
