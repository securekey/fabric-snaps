/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package relay

// MockOpts creates event relay options with the given mock event hub provider
func MockOpts(mockEventHubProvider EventHubProvider) *Opts {
	opts := DefaultOpts()
	opts.eventHubProvider = mockEventHubProvider
	return opts
}
