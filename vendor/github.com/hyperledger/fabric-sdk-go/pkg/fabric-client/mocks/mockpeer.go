/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

// TODO: Move protos to this library
import (
	"encoding/pem"
	"errors"

	api "github.com/hyperledger/fabric-sdk-go/api"

	pb "github.com/hyperledger/fabric/protos/peer"
)

// MockPeer is a mock fabricsdk.Peer.
type MockPeer struct {
	MockName  string
	MockURL   string
	MockRoles []string
	MockCert  *pem.Block
}

// ConnectEventSource does not connect anywhere
func (p *MockPeer) ConnectEventSource() {
	// done.
}

// IsEventListened always returns true
func (p *MockPeer) IsEventListened(event string, chain api.Channel) (bool, error) {
	return true, nil
}

// AddListener is not implemented
func (p *MockPeer) AddListener(eventType string, eventTypeData interface{}, eventCallback interface{}) (string, error) {
	return "", errors.New("Not implemented")
}

// RemoveListener is not implemented
func (p *MockPeer) RemoveListener(eventListenerRef string) (bool, error) {
	return false, errors.New("Not implemented")
}

// Name returns the mock peer's mock name
func (p MockPeer) Name() string {
	return p.MockName
}

// SetName sets the mock peer's mock name
func (p *MockPeer) SetName(name string) {
	p.MockName = name
}

// Roles returns the mock peer's mock roles
func (p *MockPeer) Roles() []string {
	return p.MockRoles
}

// SetRoles sets the mock peer's mock roles
func (p *MockPeer) SetRoles(roles []string) {
	p.MockRoles = roles
}

// EnrollmentCertificate returns the mock peer's mock enrollment certificate
func (p *MockPeer) EnrollmentCertificate() *pem.Block {
	return p.MockCert
}

// SetEnrollmentCertificate sets the mock peer's mock enrollment certificate
func (p *MockPeer) SetEnrollmentCertificate(pem *pem.Block) {
	p.MockCert = pem
}

// URL returns the mock peer's mock URL
func (p *MockPeer) URL() string {
	return p.MockURL
}

// SendProposal does not send anything anywhere but returns an empty mock ProposalResponse
func (p *MockPeer) SendProposal(tp *api.TransactionProposal) (*api.TransactionProposalResponse, error) {

	return &api.TransactionProposalResponse{
		Endorser:         p.MockURL,
		Proposal:         tp,
		ProposalResponse: &pb.ProposalResponse{Response: &pb.Response{Message: "success", Status: 99, Payload: []byte("")}},
	}, nil

}
