package handler

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
)

//NewInjectResponseHandler returns a handler that populates the Endorsement Response to the request context
func NewInjectResponseHandler(response *invoke.Response, next ...invoke.Handler) *InjectResponseHandler {
	return &InjectResponseHandler{response: response, next: getNext(next)}
}

//InjectResponseHandler holds the Endorsement response
type InjectResponseHandler struct {
	next     invoke.Handler
	response *invoke.Response
}

//Handle for endorsing transactions
func (i *InjectResponseHandler) Handle(requestContext *invoke.RequestContext, clientContext *invoke.ClientContext) {
	requestContext.Response = *i.response
	i.next.Handle(requestContext, clientContext)
}
