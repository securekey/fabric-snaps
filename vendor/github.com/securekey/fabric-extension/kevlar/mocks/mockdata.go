/*
   Copyright SecureKey Technologies Inc.
   This file contains software code that is the intellectual property of SecureKey.
   SecureKey reserves all rights in the code and you may not use it without
	 written permission from SecureKey.
*/

package mocks

import (
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	ledgerUtil "github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
)

// NewSimpleMockBlock returns a simple mock block
func NewSimpleMockBlock() *common.Block {
	return &common.Block{
		Data: &common.BlockData{
			Data: [][]byte{[]byte("")},
		},
	}
}

// CreateBlockWithCCEvent creates a mock block
func CreateBlockWithCCEvent(events *peer.ChaincodeEvent, txID string,
	channelID string) (*common.Block, error) {
	chdr := &common.ChannelHeader{
		Type:    int32(common.HeaderType_ENDORSER_TRANSACTION),
		Version: 1,
		Timestamp: &timestamp.Timestamp{
			Seconds: time.Now().Unix(),
			Nanos:   0,
		},
		ChannelId: channelID,
		TxId:      txID}
	hdr := &common.Header{ChannelHeader: utils.MarshalOrPanic(chdr)}
	payload := &common.Payload{Header: hdr}
	cea := &peer.ChaincodeEndorsedAction{}
	ccaPayload := &peer.ChaincodeActionPayload{Action: cea}
	env := &common.Envelope{}
	taa := &peer.TransactionAction{}
	taas := make([]*peer.TransactionAction, 1)
	taas[0] = taa
	tx := &peer.Transaction{Actions: taas}

	pHashBytes := []byte("proposal_hash")
	pResponse := &peer.Response{Status: 200}
	results := []byte("results")
	eventBytes, err := utils.GetBytesChaincodeEvent(events)
	if err != nil {
		return nil, err
	}
	ccaPayload.Action.ProposalResponsePayload, err = utils.GetBytesProposalResponsePayload(pHashBytes, pResponse, results, eventBytes, nil)
	if err != nil {
		return nil, err
	}
	tx.Actions[0].Payload, err = utils.GetBytesChaincodeActionPayload(ccaPayload)
	if err != nil {
		return nil, err
	}
	payload.Data, err = utils.GetBytesTransaction(tx)
	if err != nil {
		return nil, err
	}
	env.Payload, err = utils.GetBytesPayload(payload)
	if err != nil {
		return nil, err
	}
	ebytes, err := utils.GetBytesEnvelope(env)
	if err != nil {
		return nil, err
	}

	block := common.NewBlock(1, []byte{})
	block.Data.Data = append(block.Data.Data, ebytes)
	block.Header.DataHash = block.Data.Hash()
	txsfltr := ledgerUtil.NewTxValidationFlags(len(block.Data.Data))

	block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = txsfltr

	return block, nil
}
