/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"crypto/x509"

	"encoding/pem"

	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/pkg/errors"
	transactionsnapApi "github.com/securekey/fabric-snaps/transactionsnap/api"
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
func NewNetworkPeer(peerConfig *fabApi.PeerConfig, mspID string, pemBytes []byte) (*fabApi.NetworkPeer, error) {
	networkPeer := &fabApi.NetworkPeer{PeerConfig: *peerConfig, MSPID: mspID}

	block, _ := pem.Decode(pemBytes)
	if block != nil {
		pub, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, errors.WithMessage(err, "certificate parsing failed")
		}
		networkPeer.TLSCACert = pub
	}

	return networkPeer, nil
}
