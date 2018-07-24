/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package httpsnapservice

import (
	httpsnapApi "github.com/securekey/fabric-snaps/httpsnap/api"
	"github.com/securekey/fabric-snaps/util/errors"
)

type invoker struct {
	request      HTTPServiceInvokeRequest
	schemaConfig *httpsnapApi.SchemaConfig
	service      service
	headers      map[string]string
}

type service interface {
	getData(req HTTPServiceInvokeRequest) (responseContentType string, responseBody []byte, err errors.Error)
	validate(contentType string, schema string, body string) errors.Error
}

// Invoke invokes the HTTP service synchronously and returns the response or error
func (inv *invoker) Invoke() ([]byte, errors.Error) {
	_, response, err := inv.service.getData(inv.request)
	if err != nil {
		errorObj := errors.WithMessage(errors.GeneralError, err, "getData returned an error")
		logger.Errorf("[txID %s] %s", inv.request.TxID, errorObj.GenerateLogMsg())
		return nil, errorObj
	}

	logger.Debugf("Successfully retrieved data from URL: %s", inv.request.RequestURL)

	// Validate response body against schema
	if err := inv.service.validate(inv.headers[contentType], inv.schemaConfig.Response, string(response)); err != nil {
		errorObj := errors.WithMessage(errors.GeneralError, err, "validate returned an error")
		logger.Errorf("[txID %s] %s", inv.request.TxID, errorObj.GenerateLogMsg())
		return nil, errorObj
	}

	return response, nil
}

// InvokeAsync invokes the HTTP service asynchronously and returns a response channel and error channel,
// one of which will return the result of the invocation
func (inv *invoker) InvokeAsync() (chan []byte, chan errors.Error) {
	respChan := make(chan []byte, 1)
	errChan := make(chan errors.Error, 1)

	go func() {
		// URL is ok, retrieve data using http client
		_, response, err := inv.service.getData(inv.request)
		if err != nil {
			errorObj := errors.WithMessage(errors.GeneralError, err, "getData returned an error")
			logger.Errorf("[txID %s] %s", inv.request.TxID, errorObj.GenerateLogMsg())
			errChan <- errorObj
			return
		}

		logger.Debugf("Successfully retrieved data from URL: %s", inv.request.RequestURL)

		// Validate response body against schema
		if err := inv.service.validate(inv.headers[contentType], inv.schemaConfig.Response, string(response)); err != nil {
			errorObj := errors.WithMessage(errors.GeneralError, err, "validate returned an error")
			logger.Errorf("[txID %s] %s", inv.request.TxID, errorObj.GenerateLogMsg())
			errChan <- errorObj
			return
		}

		respChan <- response
	}()

	return respChan, errChan
}
