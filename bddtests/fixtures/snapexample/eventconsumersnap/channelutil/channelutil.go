/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package channelutil

import (
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/securekey/fabric-snaps/util/errors"
)

// ChannelIDFromFilteredBlock returns the ID of the channel for the given filtered block.
func ChannelIDFromFilteredBlock(fblock *pb.FilteredBlock) (channelID string, err error) {
	return fblock.ChannelId, nil
}

// ChannelIDFromBlock returns the ID of the channel for the given block.
func ChannelIDFromBlock(block *common.Block) (channelID string, err error) {
	if block == nil || block.Data == nil || len(block.Data.Data) == 0 {
		return "", errors.New(errors.GeneralError, "invalid block data")
	}

	data := block.Data.Data[0]
	if data == nil {
		return "", errors.New(errors.GeneralError, "invalid block data")
	}

	env, err := utils.GetEnvelopeFromBlock(data)
	if err != nil {
		return "", err
	}
	if env == nil {
		return "", errors.New(errors.GeneralError, "no envelope found in block data")
	}

	payload, err := utils.GetPayload(env)
	if err != nil {
		return "", errors.WithMessage(errors.GeneralError, err, "could not extract payload from envelope")
	}

	if payload == nil || payload.Header == nil {
		return "", errors.New(errors.GeneralError, "invalid payload")
	}

	chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return "", err
	}

	return chdr.ChannelId, nil
}
