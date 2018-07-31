/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package peerfilter

import (
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client/peerfilter/minblockheight"
	"github.com/securekey/fabric-snaps/util/errors"
)

// New creates a new peer filter according to the given options
func New(opts *api.PeerFilterOpts) (api.PeerFilter, error) {
	if opts == nil {
		return nil, nil
	}

	switch opts.Type {
	case api.MinBlockHeightPeerFilterType:
		return minblockheight.New(opts.Args)
	default:
		return nil, errors.Errorf(errors.SystemError, "invalid peer filter type [%s]", opts.Type)
	}
}
