package handler

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
)

//NewPreEndorsedHandler returns a handler that populates the Endorsement Response to the request context
func NewPreEndorsedHandler(endorserResponse *channel.Response, next ...invoke.Handler) *PreEndorsedHandler {
	return &PreEndorsedHandler{endorserResponse: endorserResponse, next: getNext(next)}
}

//PreEndorsedHandler holds the Endorsement response
type PreEndorsedHandler struct {
	next             invoke.Handler
	endorserResponse *channel.Response
}

//Handle for endorsing transactions
func (i *PreEndorsedHandler) Handle(requestContext *invoke.RequestContext, clientContext *invoke.ClientContext) {

	requestContext.Response = getResponse(i.endorserResponse)
	i.next.Handle(requestContext, clientContext)
}

func getResponse(res *channel.Response) invoke.Response {
	return invoke.Response{
		Payload:          res.Payload,
		ChaincodeStatus:  res.ChaincodeStatus,
		TransactionID:    res.TransactionID,
		Responses:        res.Responses,
		TxValidationCode: res.TxValidationCode,
		Proposal:         res.Proposal,
	}
}
