/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

//HTTPSnapRequest is used to invoke http snap
type HTTPSnapRequest struct {
	URL         string            // required
	Headers     map[string]string // required
	Body        string            // required
	NamedClient string            // optional
	PinSet      []string          // optional
}
