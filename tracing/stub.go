/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tracing

import (
	"bytes"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/securekey/fabric-snaps/util/errors"
)

// StartChildSpan will start a tracing child default span
func StartChildSpan(stub shim.ChaincodeStubInterface) opentracing.Span {
	wireContext, err := getWireContext(stub)
	if err != nil {
		return nil
	}
	fn, _ := stub.GetFunctionAndParameters()

	opts := []opentracing.StartSpanOption{opentracing.ChildOf(wireContext)}
	span := opentracing.StartSpan(fn, opts...)
	span.SetTag("TxID", stub.GetTxID())
	// TODO set Span Context somehwere...
	//stub.SetCtx(opentracing.ContextWithSpan(context.Background(), span))
	return span
}

// StartFollowSpan will start a tracing span that follows a previous span (not parent), example: FMP Message
func StartFollowSpan(stub shim.ChaincodeStubInterface) opentracing.Span {
	wireContext, err := getWireContext(stub)
	if err != nil {
		return nil
	}
	fn, _ := stub.GetFunctionAndParameters()

	opts := []opentracing.StartSpanOption{opentracing.FollowsFrom(wireContext), ext.SpanKindConsumer}
	span := opentracing.StartSpan(fn, opts...)
	span.SetTag("TxID", stub.GetTxID())
	// TODO set Span Context somehwere...
	//stub.SetCtx(opentracing.ContextWithSpan(context.Background(), span))
	return span
}

// StartRPCServerSpan will start a tracing span for an RPC message from the server side
func StartRPCServerSpan(stub shim.ChaincodeStubInterface) opentracing.Span {
	return startChildTaggedSpan(stub, ext.SpanKindRPCServer)
}

// StartRPCClientSpan will start a tracing span for an RPC message from the client side
func StartRPCClientSpan(stub shim.ChaincodeStubInterface) opentracing.Span {
	return startChildTaggedSpan(stub, ext.SpanKindRPCClient)
}

func startChildTaggedSpan(stub shim.ChaincodeStubInterface, spanKind opentracing.Tag) opentracing.Span {
	wireContext, err := getWireContext(stub)
	if err != nil {
		return nil
	}
	fn, _ := stub.GetFunctionAndParameters()

	opts := []opentracing.StartSpanOption{opentracing.ChildOf(wireContext), spanKind}
	span := opentracing.StartSpan(fn, opts...)
	span.SetTag("TxID", stub.GetTxID())
	// TODO set Span Context somehwere...
	//stub.SetCtx(opentracing.ContextWithSpan(context.Background(), span))
	return span
}

func getWireContext(stub shim.ChaincodeStubInterface) (opentracing.SpanContext, error) {
	transientMap, err := stub.GetTransient()
	if err != nil {
		return nil, err
	}
	spanCtx, ok := transientMap["spanCtx"]
	if !ok {
		return nil, errors.New(errors.GeneralError, "spanCtx missing in transientMap")
	}

	read := bytes.NewReader(spanCtx)
	return opentracing.GlobalTracer().Extract(opentracing.Binary, read)
}

// SpanFromStub retrieves tracing span from stub/context.. TODO to be implemented
//func SpanFromStub(stub shim.ChaincodeStubInterface) opentracing.Span {
//	ctx := context.TODO()
//	vmeStub, err := ToVmeChaincodeStub(stub)
//	if err != nil {
//		myLogger.Warn(logwarning.General, "failed to build VmeChaincodeStub")
//	} else {
//		ctx = vmeStub.GetCtx()
//	}
//	return opentracing.SpanFromContext(ctx)
//}
