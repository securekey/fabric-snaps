/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

// PublicKeyForLogging is public key and key id combination used for private logging
type PublicKeyForLogging struct {
	// PublicKey is the public key used for private logging
	PublicKey string `json:"publickey,omitempty"`

	// KeyID is the key ID used for private logging
	KeyID string `json:"keyid,omitempty"`
}
