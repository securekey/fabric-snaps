/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/hyperledger/fabric/msp"
	mp "github.com/hyperledger/fabric/protos/msp"
)

// MSP implements mock msp
type MSP struct {
	err      error
	identity msp.Identity
}

// NewMSP creates mock msp
func NewMSP() *MSP {
	return &MSP{}
}

// WithError injects an error
func (m *MSP) WithError(err error) *MSP {
	m.err = err
	return m
}

// WithIdentity injects an error on the deserialized identity
func (m *MSP) WithIdentity(identity msp.Identity) *MSP {
	m.identity = identity
	return m
}

// DeserializeIdentity mockcore deserialize identity
func (m *MSP) DeserializeIdentity(serializedIdentity []byte) (msp.Identity, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.identity != nil {
		return m.identity, nil
	}
	return &Identity{}, nil
}

// IsWellFormed  checks if the given identity can be deserialized into its provider-specific form
func (m *MSP) IsWellFormed(identity *mp.SerializedIdentity) error {
	return m.err
}

// Setup the MSP instance according to configuration information
func (m *MSP) Setup(config *mp.MSPConfig) error {
	return m.err
}

// GetMSPs Provides a list of Membership Service providers
func (m *MSP) GetMSPs() (map[string]msp.MSP, error) {
	return nil, m.err
}

// GetVersion returns the version of this MSP
func (m *MSP) GetVersion() msp.MSPVersion {
	return 0
}

// GetType returns the provider type
func (m *MSP) GetType() msp.ProviderType {
	return 0
}

// GetIdentifier returns the provider identifier
func (m *MSP) GetIdentifier() (string, error) {
	return "", nil
}

// GetSigningIdentity returns a signing identity corresponding to the provided identifier
func (m *MSP) GetSigningIdentity(identifier *msp.IdentityIdentifier) (msp.SigningIdentity, error) {
	return nil, m.err
}

// GetDefaultSigningIdentity returns the default signing identity
func (m *MSP) GetDefaultSigningIdentity() (msp.SigningIdentity, error) {
	return nil, m.err
}

// GetTLSRootCerts returns the TLS root certificates for this MSP
func (m *MSP) GetTLSRootCerts() [][]byte {
	return nil
}

// GetTLSIntermediateCerts returns the TLS intermediate root certificates for this MSP
func (m *MSP) GetTLSIntermediateCerts() [][]byte {
	return nil
}

// Validate checks whether the supplied identity is valid
func (m *MSP) Validate(id msp.Identity) error {
	return m.err
}

// SatisfiesPrincipal checks whether the identity matches
// the description supplied in MSPPrincipal.
func (m *MSP) SatisfiesPrincipal(id msp.Identity, principal *mp.MSPPrincipal) error {
	return m.err
}
