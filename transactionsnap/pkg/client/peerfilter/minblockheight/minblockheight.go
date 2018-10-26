/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package minblockheight

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	memserviceapi "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	transactionsnapApi "github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("txnsnap")

// New creates a new Min Block Height peer filter. This filter
// selects peers whose block height is at least that of the
// provided value.
// - arg[0] - Channel ID
// - arg[1] - Minimum block height
func New(args []string) (transactionsnapApi.PeerFilter, error) {
	if len(args) < 2 {
		return nil, errors.New(errors.SystemError, "expecting channel ID and block height args")
	}

	height, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return nil, errors.WithMessage(errors.SystemError, err, "invalid block height arg "+args[1])
	}

	service, err := client.MemServiceProvider()
	if err != nil {
		return nil, errors.WithMessage(errors.SystemError, err, "error getting membership service")
	}

	logger.Debugf("Creating MinBlockHeight peer filter - Channel [%s], Height [%d]", args[0], height)

	return &peerFilter{
		channelID: args[0],
		height:    height,
		service:   service,
	}, nil
}

type peerFilter struct {
	channelID string
	height    uint64
	service   memserviceapi.Service
}

// Accept returns true if the given peer's block height is
// at least the height of the required minimum.
func (f *peerFilter) Accept(p fabApi.Peer) bool {
	peerHeight := f.getBlockHeight(p)
	accepted := peerHeight >= f.height
	if !accepted {
		logger.Debugf("Peer [%s] will NOT be accepted since its block height for channel [%s] is %d which is less than the required minimum: %d.", p.URL(), f.channelID, peerHeight, f.height)
	} else {
		logger.Debugf("Peer [%s] will be accepted since its block height for channel [%s] is %d which is greater than or equal to that of the required minimum: %d.", p.URL(), f.channelID, peerHeight, f.height)
	}

	return accepted
}

func (f *peerFilter) getBlockHeight(p fabApi.Peer) uint64 {
	endpoints, err := f.service.GetPeersOfChannel(f.channelID)
	if err != nil {
		logger.Errorf(errors.WithMessage(errors.SystemError, err, fmt.Sprintf("Error querying for peers of channel [%s]", f.channelID)).GenerateLogMsg())
		return 0
	}

	for _, endpoint := range endpoints {
		// p.Url() will be in the for grpc://host:port whereas
		// the endpoint will be in the form host:port
		if strings.Contains(p.URL(), endpoint.Endpoint) {
			logger.Debugf("Block height for [%s] in channel [%s]: %d", p.URL(), f.channelID, endpoint.LedgerHeight)
			return endpoint.LedgerHeight
		}
	}

	logger.Warnf("Peer [%s] not found for channel [%s]", p.URL(), f.channelID)

	return 0
}
