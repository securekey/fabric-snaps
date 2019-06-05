/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorse

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
)

// RequestOptions opts allows the user to specify more advanced options
type RequestOptions struct {
	Handler invoke.Handler
}

// RequestOption func for each Opts argument
type RequestOption func(opts *RequestOptions) error

// WithHandler set custom handler
func WithHandler(handler invoke.Handler) RequestOption {
	return func(o *RequestOptions) error {
		o.Handler = handler
		return nil
	}
}
