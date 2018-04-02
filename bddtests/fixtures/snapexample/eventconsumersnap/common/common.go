/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

// ByteFilteredBlockEvent represents a FilteredBlockEvent jsonpb marshalled FilteredBlock objects
type ByteFilteredBlockEvent struct {
	// Payload is the jsonpb bytes representation of FilteredBlock from FilteredBlockEvent
	Payload []byte
	// SourceURL specifies the URL of the peer that produced the event
	SourceURL string
}
