/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

const (
	// ConfigCCEventName is the name of the chaincode event that is published to
	// indicate that the configuration has changed
	ConfigCCEventName = "cfgsnap-event"

	//GeneralMspID is the msp id of generic config
	GeneralMspID = "general"
)

// PublicKeyForLogging is public key and key id combination used for private logging
type PublicKeyForLogging struct {
	// PublicKey is the public key used for private logging
	PublicKey string `json:"publickey,omitempty"`

	// KeyID is the key ID used for private logging
	KeyID string `json:"keyid,omitempty"`
}
