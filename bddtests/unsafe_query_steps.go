/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"fmt"
	"strings"

	"github.com/DATA-DOG/godog"
	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

// UnsafeQuerySteps unsafe query BDD test steps
type UnsafeQuerySteps struct {
	BDDContext *BDDContext
}

// NewUnsafeQuerySteps new unsafe query steps
func NewUnsafeQuerySteps(context *BDDContext) *UnsafeQuerySteps {
	return &UnsafeQuerySteps{BDDContext: context}
}

// InvokeCCVerifyResponse invoke CC and verify unsafe query operation
func (u *UnsafeQuerySteps) InvokeCCVerifyResponse(ccID, userArgs, orgIDs, channelID, expectedResponse string) error {
	commonSteps := NewCommonSteps(u.BDDContext)
	targets := commonSteps.OrgPeers(orgIDs, channelID)
	args := strings.Split(userArgs, ",")
	if len(targets) == 0 {
		return fmt.Errorf("no target peer specified")
	}
	target := targets[0]

	targetPeer, err := u.BDDContext.OrgUserContext(targets[0].OrgID, ADMIN).InfraProvider().CreatePeerFromConfig(&fabApi.NetworkPeer{PeerConfig: target.Config})
	if err != nil {
		return errors.WithMessage(err, "NewPeer failed")
	}

	chClient, err := u.BDDContext.OrgChannelClient(targets[0].OrgID, USER, channelID)
	if err != nil {
		return fmt.Errorf("Failed to create new channel client: %s", err)
	}

	resp, err := chClient.Execute(
		channel.Request{
			ChaincodeID: ccID,
			Fcn:         args[0],
			Args:        GetByteArgs(args[1:]),
		}, channel.WithTargets([]fabApi.Peer{targetPeer}...))
	if err != nil {
		return fmt.Errorf("InvokeChaincode return error: %v", err)
	}

	if string(resp.Payload) != expectedResponse {
		return fmt.Errorf("Response payload did not match. Expected %s, got %s", expectedResponse, string(resp.Payload))
	}

	reads, err := getReadSet(resp.Responses[0].ProposalResponse)
	if err != nil {
		return fmt.Errorf("Could not extract read set from response: %s", err.Error())
	}

	if len(reads) != 0 {
		return fmt.Errorf("Reads were present in transaction %+v", reads)
	}

	return nil
}

func getReadSet(r *pb.ProposalResponse) ([]*kvrwset.KVRead, error) {
	reads := []*kvrwset.KVRead{}

	prp := &pb.ProposalResponsePayload{}
	if err := proto.Unmarshal(r.Payload, prp); err != nil {
		return reads, errors.WithMessage(err, "Error unmarshaling to ProposalResponsePayload")
	}

	ccAction := &pb.ChaincodeAction{}
	if err := proto.Unmarshal(prp.Extension, ccAction); err != nil {
		return reads, errors.WithMessage(err, "Error unmarshaling to ChaincodeAction")
	}

	txRWSet := &rwsetutil.TxRwSet{}
	if err := txRWSet.FromProtoBytes(ccAction.Results); err != nil {
		return reads, errors.WithMessage(err, "Error unmarshaling to txRWSet")
	}

	for _, nsRWSet := range txRWSet.NsRwSets {
		// Skip reads from lscc
		if nsRWSet.NameSpace == "lscc" {
			continue
		}
		if nsRWSet.KvRwSet != nil && len(nsRWSet.KvRwSet.Reads) > 0 {
			logger.Infof("Found Read on key %v from namespace %s", nsRWSet.KvRwSet.Reads[0].Key, nsRWSet.NameSpace)
			reads = append(reads, nsRWSet.KvRwSet.Reads...)
		}
	}
	return reads, nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (u *UnsafeQuerySteps) registerSteps(s *godog.Suite) {
	s.BeforeScenario(u.BDDContext.BeforeScenario)
	s.AfterScenario(u.BDDContext.AfterScenario)
	s.Step(`^client invokes chaincode "([^"]*)" with args "([^"]*)" on a peer in the "([^"]*)" org on the "([^"]*)" channel it gets response "([^"]*)" and the read set is empty$`, u.InvokeCCVerifyResponse)
}
