package handler

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
)

//NewValidateEndorsementHandler returns a handler that validate endorsement
func NewValidateEndorsementHandler(proposalResponses []*pb.ProposalResponse, next ...invoke.Handler) *ValidateEndorsementHandler {
	return &ValidateEndorsementHandler{proposalResponses: proposalResponses, next: getNext(next)}
}

//ValidateEndorsementHandler holds the endorsement
type ValidateEndorsementHandler struct {
	next              invoke.Handler
	proposalResponses []*pb.ProposalResponse
}

//Handle for endorsing transactions
func (i *ValidateEndorsementHandler) Handle(requestContext *invoke.RequestContext, clientContext *invoke.ClientContext) {
	//TODO validate there are enough endorsements that match chain code policy
	//TODO validate the endorsements are for the specific channel

	var responses []*fab.TransactionProposalResponse
	for _, proposalResponse := range i.proposalResponses {
		responses = append(responses, &fab.TransactionProposalResponse{ProposalResponse: proposalResponse})
	}

	requestContext.Response.Responses = responses
	if len(i.proposalResponses) > 0 {
		requestContext.Response.Payload = i.proposalResponses[0].GetResponse().Payload
		requestContext.Response.ChaincodeStatus = i.proposalResponses[0].Response.Status
	}

	i.next.Handle(requestContext, clientContext)
}
