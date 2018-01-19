/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package peerfilter

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/client/peerfilter/minblockheight"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("txnsnap")

// New creates a new peer filter according to the given options
func New(opts *api.PeerFilterOpts) (api.PeerFilter, error) {
	if opts == nil {
		return nil, nil
	}

	switch opts.Type {
	case api.MinBlockHeightPeerFilterType:
		return minblockheight.New(opts.Args)
	default:
		return nil, errors.Errorf(errors.GeneralError, "invalid peer filter type [%s]", opts.Type)
	}
}
