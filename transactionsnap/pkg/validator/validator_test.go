/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"fmt"
	"testing"

	"github.com/hyperledger/fabric/core/ledger"
	cb "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ccLSCC    = "lscc"
	channelID = "testchannel"

	ccID1 = "chaincode1"
	ccID2 = "chaincode2"
	ccID3 = "chaincode3"

	ccVersion1 = "v1"
	ccVersion2 = "v2"

	org1MSP = "Org1MSP"
	org2MSP = "Org2MSP"
)

func TestValidator_ValidateProposalResponses_InvalidInput(t *testing.T) {
	cdBuilder := mocks.NewChaincodeDataBuilder().
		Name(ccID1).
		Version(ccVersion1).
		VSCC("vscc").
		Policy("OutOf(1,'Org1MSP.member')")

	qe := mocks.NewQueryExecutor().State(ccLSCC, ccID1, cdBuilder.BuildBytes())
	identity := mocks.NewIdentity()
	msp := mocks.NewMSP().WithIdentity(identity)

	v := newValidator(channelID, newQueryExecutorProvider(qe), newMockPolicyEvaluator(), msp)
	require.NotNil(t, v)

	t.Run("No Proposal Responses -> error", func(t *testing.T) {
		proposal := &pb.SignedProposal{}
		var responses []*pb.ProposalResponse

		code, err := v.ValidateProposalResponses(proposal, responses)
		assert.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		assert.EqualError(t, err, "no proposal responses")
	})

	t.Run("Nil chaincode response -> error", func(t *testing.T) {
		pBuilder := mocks.NewProposalBuilder().ChannelID(channelID).MSPID(org1MSP).ChaincodeID(ccID1)
		rBuilder := mocks.NewProposalResponsesBuilder()
		rBuilder.ProposalResponse()

		code, err := v.ValidateProposalResponses(pBuilder.Build(), rBuilder.Build())
		assert.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		assert.EqualError(t, err, "nil chaincode response")
	})

	t.Run("Failed chaincode response status -> error", func(t *testing.T) {
		pBuilder := mocks.NewProposalBuilder().ChannelID(channelID).MSPID(org1MSP).ChaincodeID(ccID1)
		rBuilder := mocks.NewProposalResponsesBuilder()
		rBuilder.ProposalResponse().Response().Status(int32(cb.Status_BAD_REQUEST))

		code, err := v.ValidateProposalResponses(pBuilder.Build(), rBuilder.Build())
		assert.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		assert.EqualError(t, err, "failed chaincode response status")
	})

	t.Run("No endorsement in response -> error", func(t *testing.T) {
		pBuilder := mocks.NewProposalBuilder().ChaincodeID(ccID1)

		responsesBuilder := mocks.NewProposalResponsesBuilder()
		rBuilder := responsesBuilder.ProposalResponse()
		rBuilder.Response().Status(int32(cb.Status_SUCCESS))
		rBuilder.Payload().ChaincodeAction().ChaincodeID(ccID2, "", ccVersion1)

		code, err := v.ValidateProposalResponses(pBuilder.Build(), responsesBuilder.Build())
		assert.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		assert.EqualError(t, err, "missing endorsement in proposal response")
	})

	t.Run("Nil chaincode ID in ChaincodeAction -> error", func(t *testing.T) {
		pBuilder := mocks.NewProposalBuilder().ChannelID(channelID).MSPID(org1MSP).ChaincodeID(ccID1)
		responsesBuilder := mocks.NewProposalResponsesBuilder()
		rBuilder := responsesBuilder.ProposalResponse()
		rBuilder.Response().Status(int32(cb.Status_SUCCESS))
		rBuilder.Endorsement()

		code, err := v.ValidateProposalResponses(pBuilder.Build(), responsesBuilder.Build())
		assert.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		assert.EqualError(t, err, "nil ChaincodeId in ChaincodeAction")
	})

	t.Run("Invalid chaincode version -> error", func(t *testing.T) {
		pBuilder := mocks.NewProposalBuilder().ChannelID(channelID).MSPID(org1MSP).ChaincodeID(ccID1)
		responsesBuilder := mocks.NewProposalResponsesBuilder()
		rBuilder := responsesBuilder.ProposalResponse()
		rBuilder.Response().Status(int32(cb.Status_SUCCESS))
		rBuilder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", "")
		rBuilder.Endorsement()

		code, err := v.ValidateProposalResponses(pBuilder.Build(), responsesBuilder.Build())
		assert.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		assert.EqualError(t, err, "invalid chaincode version in ChaincodeAction")
	})

	t.Run("Invalid chaincode ID in proposal -> error", func(t *testing.T) {
		pBuilder := mocks.NewProposalBuilder()

		responsesBuilder := mocks.NewProposalResponsesBuilder()
		rBuilder := responsesBuilder.ProposalResponse()
		rBuilder.Response().Status(int32(cb.Status_SUCCESS))
		rBuilder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		rBuilder.Endorsement()

		code, err := v.ValidateProposalResponses(pBuilder.Build(), responsesBuilder.Build())
		assert.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		assert.EqualError(t, err, "invalid chaincode ID in proposal")
	})

	t.Run("inconsistent chaincode ID info -> error", func(t *testing.T) {
		pBuilder := mocks.NewProposalBuilder().ChaincodeID(ccID1)

		responsesBuilder := mocks.NewProposalResponsesBuilder()
		rBuilder := responsesBuilder.ProposalResponse()
		rBuilder.Response().Status(int32(cb.Status_SUCCESS))
		rBuilder.Payload().ChaincodeAction().ChaincodeID(ccID2, "", ccVersion1)
		rBuilder.Endorsement()

		code, err := v.ValidateProposalResponses(pBuilder.Build(), responsesBuilder.Build())
		assert.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		assert.EqualError(t, err, "inconsistent chaincode ID info (chaincode1/chaincode2)")
	})

	t.Run("Chaincode event chaincode ID does not match -> error", func(t *testing.T) {
		pBuilder := mocks.NewProposalBuilder().ChaincodeID(ccID1)

		responsesBuilder := mocks.NewProposalResponsesBuilder()
		rBuilder := responsesBuilder.ProposalResponse()
		rBuilder.Response().Status(int32(cb.Status_SUCCESS))
		rBuilder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1).Events().ChaincodeID(ccID2)
		rBuilder.Endorsement()

		code, err := v.ValidateProposalResponses(pBuilder.Build(), responsesBuilder.Build())
		assert.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		assert.EqualError(t, err, "chaincode event chaincode id does not match chaincode action chaincode id")
	})

	t.Run("Endorsement mismatch -> error", func(t *testing.T) {
		pBuilder := mocks.NewProposalBuilder().ChaincodeID(ccID1)

		responsesBuilder := mocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS))
		r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		r1Builder.Endorsement()
		r2Builder := responsesBuilder.ProposalResponse()
		r2Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("some payload"))
		r2Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		r2Builder.Endorsement()

		code, err := v.ValidateProposalResponses(pBuilder.Build(), responsesBuilder.Build())
		assert.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		assert.EqualError(t, err, "one or more proposal responses do not match")
	})

	t.Run("Invalid signature -> error", func(t *testing.T) {
		identity.WithError(fmt.Errorf("invalid signature"))
		pBuilder := mocks.NewProposalBuilder().ChaincodeID(ccID1)

		responsesBuilder := mocks.NewProposalResponsesBuilder()
		rBuilder := responsesBuilder.ProposalResponse()
		rBuilder.Response().Status(int32(cb.Status_SUCCESS))
		rBuilder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		rBuilder.Endorsement()

		code, err := v.ValidateProposalResponses(pBuilder.Build(), responsesBuilder.Build())
		assert.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		assert.EqualError(t, err, "the creator certificate is not valid: invalid signature")
	})
}

func TestValidator_ValidateProposalResponses_ChaincodeDataError(t *testing.T) {
	pBuilder := mocks.NewProposalBuilder().ChannelID(channelID).MSPID(org1MSP).ChaincodeID(ccID1)

	qe := mocks.NewQueryExecutor()
	identity1 := mocks.NewIdentity().WithMSPID(org1MSP)
	msp := mocks.NewMSP()

	v := newValidator(channelID, newQueryExecutorProvider(qe), newMockPolicyEvaluator(), msp)
	require.NotNil(t, v)

	t.Run("Get chaincode data -> error", func(t *testing.T) {
		id1Bytes, err := identity1.Serialize()
		require.NoError(t, err)

		responsesBuilder := mocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Endorsement().Endorser(id1Bytes).Signature([]byte("id1"))
		ca1Builder := r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		ca1Builder.Results().Namespace(ccID2).Write("key1", []byte("value1"))

		code, err := v.ValidateProposalResponses(pBuilder.Build(), responsesBuilder.Build())
		assert.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Could not retrieve state for chaincode")
	})

	t.Run("No VSCC in chaincode data -> error", func(t *testing.T) {
		cdBuilder := mocks.NewChaincodeDataBuilder().
			Name(ccID1).
			Version(ccVersion1).
			Policy("OutOf(1,'Org1MSP.member')")
		qe.State(ccLSCC, ccID1, cdBuilder.BuildBytes())

		id1Bytes, err := identity1.Serialize()
		require.NoError(t, err)

		responsesBuilder := mocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Endorsement().Endorser(id1Bytes).Signature([]byte("id1"))
		ca1Builder := r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		ca1Builder.Results().Namespace(ccID2).Write("key1", []byte("value1"))

		code, err := v.ValidateProposalResponses(pBuilder.Build(), responsesBuilder.Build())
		assert.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "vscc field must be set")
	})

	t.Run("No policy in chaincode data -> error", func(t *testing.T) {
		cdBuilder := mocks.NewChaincodeDataBuilder().
			Name(ccID1).
			Version(ccVersion1).
			VSCC("vscc")
		qe.State(ccLSCC, ccID1, cdBuilder.BuildBytes())

		id1Bytes, err := identity1.Serialize()
		require.NoError(t, err)

		responsesBuilder := mocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Endorsement().Endorser(id1Bytes).Signature([]byte("id1"))
		ca1Builder := r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		ca1Builder.Results().Namespace(ccID2).Write("key1", []byte("value1"))

		code, err := v.ValidateProposalResponses(pBuilder.Build(), responsesBuilder.Build())
		assert.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "policy field must be set")
	})

	t.Run("Expired chaincode -> error", func(t *testing.T) {
		cdBuilder := mocks.NewChaincodeDataBuilder().
			Name(ccID1).
			Version(ccVersion2).
			VSCC("vscc").
			Policy("OutOf(1,'Org1MSP.member')")
		qe.State(ccLSCC, ccID1, cdBuilder.BuildBytes())

		id1Bytes, err := identity1.Serialize()
		require.NoError(t, err)

		responsesBuilder := mocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Endorsement().Endorser(id1Bytes).Signature([]byte("id1"))
		ca1Builder := r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		ca1Builder.Results().Namespace(ccID2).Write("key1", []byte("value1"))

		code, err := v.ValidateProposalResponses(pBuilder.Build(), responsesBuilder.Build())
		assert.Equal(t, pb.TxValidationCode_EXPIRED_CHAINCODE, code)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "chaincode chaincode1:v1/testchannel didn't match chaincode1:v2/testchannel")
	})
}

func TestValidator_ValidateProposalResponses(t *testing.T) {
	pBuilder := mocks.NewProposalBuilder().ChannelID(channelID).MSPID(org1MSP).ChaincodeID(ccID1)

	qe := mocks.NewQueryExecutor()
	identity1 := mocks.NewIdentity().WithMSPID(org1MSP)
	identity2 := mocks.NewIdentity().WithMSPID(org2MSP)

	policyEvaluator := newMockPolicyEvaluator()
	msp := mocks.NewMSP()
	v := newValidator(channelID, newQueryExecutorProvider(qe), policyEvaluator, msp)
	require.NotNil(t, v)

	t.Run("Duplicate identities -> error", func(t *testing.T) {
		cdBuilder := mocks.NewChaincodeDataBuilder().
			Name(ccID1).
			Version(ccVersion1).
			VSCC("vscc").
			Policy("OutOf(2,'Org1MSP.member','Org2MSP.member')")

		qe.State(ccLSCC, ccID1, cdBuilder.BuildBytes())

		policyBytes := cdBuilder.Build().Policy
		policyEvaluator.WithError(policyBytes, fmt.Errorf("signature set did not satisfy policy"))
		defer func() { policyEvaluator.WithError(policyBytes, nil) }()

		responsesBuilder := mocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		r1Builder.Endorsement()
		r2Builder := responsesBuilder.ProposalResponse()
		r2Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r2Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		r2Builder.Endorsement()

		code, err := v.ValidateProposalResponses(pBuilder.Build(), responsesBuilder.Build())
		assert.Equal(t, pb.TxValidationCode_ENDORSEMENT_POLICY_FAILURE, code)
		assert.EqualError(t, err, errDuplicateIdentity)
	})

	t.Run("Policy satisfied -> success", func(t *testing.T) {
		cdBuilder := mocks.NewChaincodeDataBuilder().
			Name(ccID1).
			Version(ccVersion1).
			VSCC("vscc").
			Policy("OutOf(2,'Org1MSP.member','Org2MSP.member')")
		qe.State(ccLSCC, ccID1, cdBuilder.BuildBytes())

		id1Bytes, err := identity1.Serialize()
		require.NoError(t, err)
		id2Bytes, err := identity2.Serialize()
		require.NoError(t, err)

		responsesBuilder := mocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		r1Builder.Endorsement().Endorser(id1Bytes).Signature([]byte("id1"))
		r2Builder := responsesBuilder.ProposalResponse()
		r2Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r2Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		r2Builder.Endorsement().Endorser(id2Bytes).Signature([]byte("id2"))

		code, err := v.ValidateProposalResponses(pBuilder.Build(), responsesBuilder.Build())
		assert.Equal(t, pb.TxValidationCode_VALID, code)
		assert.NoError(t, err)
	})

	t.Run("CC-to-CC -> error", func(t *testing.T) {
		cd1Builder := mocks.NewChaincodeDataBuilder().
			Name(ccID1).
			Version(ccVersion1).
			VSCC("vscc").
			Policy("OutOf(1,'Org1MSP.member')")
		qe.State(ccLSCC, ccID1, cd1Builder.BuildBytes())

		cd2Builder := mocks.NewChaincodeDataBuilder().
			Name(ccID2).
			Version(ccVersion1).
			VSCC("vscc").
			Policy("OutOf(2,'Org1MSP.member','Org2MSP.member')")
		qe.State(ccLSCC, ccID2, cd2Builder.BuildBytes())

		// Set an error on CC2 policy evaluation
		policyBytes := cd2Builder.Build().Policy
		policyEvaluator.WithError(policyBytes, fmt.Errorf("signature set did not satisfy policy"))
		defer func() { policyEvaluator.WithError(policyBytes, nil) }()

		id1Bytes, err := identity1.Serialize()
		require.NoError(t, err)

		responsesBuilder := mocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Endorsement().Endorser(id1Bytes).Signature([]byte("id1"))
		ca1Builder := r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		resultsBuilder := ca1Builder.Results()
		resultsBuilder.Namespace(ccID1).Read("key1", 1000, 1)
		resultsBuilder.Namespace(ccID2).Write("key1", []byte("value1"))

		code, err := v.ValidateProposalResponses(pBuilder.Build(), responsesBuilder.Build())
		assert.Equal(t, pb.TxValidationCode_ENDORSEMENT_POLICY_FAILURE, code)
		assert.EqualError(t, err, "VSCC error: endorsement policy failure, err: signature set did not satisfy policy")
	})

	t.Run("CC-to-CC -> success", func(t *testing.T) {
		cd1Builder := mocks.NewChaincodeDataBuilder().
			Name(ccID1).
			Version(ccVersion1).
			VSCC("vscc").
			Policy("OutOf(1,'Org1MSP.member')")
		qe.State(ccLSCC, ccID1, cd1Builder.BuildBytes())

		cd2Builder := mocks.NewChaincodeDataBuilder().
			Name(ccID2).
			Version(ccVersion1).
			VSCC("vscc").
			Policy("OutOf(2,'Org1MSP.member','Org2MSP.member')")
		qe.State(ccLSCC, ccID2, cd2Builder.BuildBytes())

		cd3Builder := mocks.NewChaincodeDataBuilder().
			Name(ccID3).
			Version(ccVersion1).
			VSCC("vscc").
			Policy("OutOf(2,'Org1MSP.member','Org2MSP.member')")
		qe.State(ccLSCC, ccID3, cd3Builder.BuildBytes())

		id1Bytes, err := identity1.Serialize()
		require.NoError(t, err)

		responsesBuilder := mocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Endorsement().Endorser(id1Bytes).Signature([]byte("id1"))
		ca1Builder := r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		resultsBuilder := ca1Builder.Results()
		resultsBuilder.Namespace(ccID1).Read("key1", 1000, 1)
		resultsBuilder.Namespace(ccID2).Write("key1", []byte("value1"))

		code, err := v.ValidateProposalResponses(pBuilder.Build(), responsesBuilder.Build())
		assert.Equal(t, pb.TxValidationCode_VALID, code)
		assert.NoError(t, err)
	})

	t.Run("Write to LSCC -> error", func(t *testing.T) {
		cd1Builder := mocks.NewChaincodeDataBuilder().
			Name(ccID1).
			Version(ccVersion1).
			VSCC("vscc").
			Policy("OutOf(1,'Org1MSP.member')")
		qe.State(ccLSCC, ccID1, cd1Builder.BuildBytes())

		id1Bytes, err := identity1.Serialize()
		require.NoError(t, err)

		responsesBuilder := mocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Endorsement().Endorser(id1Bytes).Signature([]byte("id1"))
		ca1Builder := r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		ca1Builder.Results().Namespace(ccLSCC).Write("key1", []byte("value1"))

		code, err := v.ValidateProposalResponses(pBuilder.Build(), responsesBuilder.Build())
		assert.Equal(t, pb.TxValidationCode_ILLEGAL_WRITESET, code)
		assert.EqualError(t, err, "chaincode chaincode1 attempted to write to the namespace of LSCC")
	})
}

type qeProvider struct {
	qe  *mocks.QueryExecutor
	err error
}

func newQueryExecutorProvider(qe *mocks.QueryExecutor) *qeProvider {
	return &qeProvider{
		qe: qe,
	}
}

func (p *qeProvider) NewQueryExecutor() (ledger.QueryExecutor, error) {
	return p.qe, nil
}

type mockPolicyEvaluator struct {
	err map[string]error
}

func newMockPolicyEvaluator() *mockPolicyEvaluator {
	return &mockPolicyEvaluator{
		err: make(map[string]error),
	}
}

func (pe *mockPolicyEvaluator) WithError(policyBytes []byte, err error) *mockPolicyEvaluator {
	pe.err[string(policyBytes)] = err
	return pe
}

func (pe *mockPolicyEvaluator) Evaluate(policyBytes []byte, signatureSet []*cb.SignedData) error {
	return pe.err[string(policyBytes)]
}
