/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"sync"

	"fmt"

	"github.com/golang/protobuf/proto"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	rwsetutil "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	kvrwset "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	msp "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/bccsp/utils"
)

// MockPeer is a mock fabricsdk.Peer.
type MockPeer struct {
	RWLock               *sync.RWMutex
	Error                error
	MockName             string
	MockURL              string
	MockRoles            []string
	MockCert             *pem.Block
	Payload              map[string][]byte
	ResponseMessage      string
	MockMSP              string
	Status               int32
	ProcessProposalCalls int
	Endorser             []byte
	KVWrite              bool
}

// NewMockPeer creates basic mock peer
func NewMockPeer(name string, url string) *MockPeer {
	mp := &MockPeer{MockName: name, MockURL: url, Status: 200, RWLock: &sync.RWMutex{}}
	return mp
}

// Name returns the mock peer's mock name
func (p MockPeer) Name() string {
	return p.MockName
}

// SetName sets the mock peer's mock name
func (p *MockPeer) SetName(name string) {
	p.MockName = name
}

// MSPID gets the Peer mspID.
func (p *MockPeer) MSPID() string {
	return p.MockMSP
}

// SetMSPID sets the Peer mspID.
func (p *MockPeer) SetMSPID(mspID string) {
	p.MockMSP = mspID
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

// ProcessTransactionProposal does not send anything anywhere but returns an empty mock ProposalResponse
func (p *MockPeer) ProcessTransactionProposal(tp fabApi.TransactionProposal, funcName []byte) (*fabApi.TransactionProposalResponse, error) {
	if p.RWLock != nil {
		p.RWLock.Lock()
		defer p.RWLock.Unlock()
	}
	p.ProcessProposalCalls++

	if p.Endorser == nil {
		// We serialize identities by prepending the MSPID and appending the ASN.1 DER content of the cert
		sID := &msp.SerializedIdentity{Mspid: "Org1MSP", IdBytes: []byte(CertPem)}
		endorser, err := proto.Marshal(sID)
		if err != nil {
			return nil, err
		}
		p.Endorser = endorser
	}

	block, _ := pem.Decode(KeyPem)
	lowLevelKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	proposalResponsePayload, err := p.createProposalResponsePayload()
	if err != nil {
		return nil, err
	}
	sigma, err := SignECDSA(lowLevelKey, append(proposalResponsePayload, p.Endorser...))
	if err != nil {
		return nil, err
	}

	payload, ok := p.Payload[string(funcName)]
	if !ok {
		payload, ok = p.Payload[string("default")]
		if !ok {
			fmt.Printf("payload for func(%s) not found\n", funcName)
		}
	}
	return &fabApi.TransactionProposalResponse{
		Endorser: p.MockURL,
		Status:   p.Status,
		ProposalResponse: &pb.ProposalResponse{Response: &pb.Response{
			Message: p.ResponseMessage, Status: p.Status, Payload: payload}, Payload: proposalResponsePayload,
			Endorsement: &pb.Endorsement{Endorser: p.Endorser, Signature: sigma}},
	}, p.Error

}

func (p *MockPeer) createProposalResponsePayload() ([]byte, error) {

	prp := &pb.ProposalResponsePayload{}
	ccAction := &pb.ChaincodeAction{}
	txRwSet := &rwsetutil.TxRwSet{}
	var kvWrite []*kvrwset.KVWrite
	if p.KVWrite {
		kvWrite = []*kvrwset.KVWrite{{Key: "key2", IsDelete: false, Value: []byte("value2")}}
	}
	txRwSet.NsRwSets = []*rwsetutil.NsRwSet{
		{NameSpace: "ns1", KvRwSet: &kvrwset.KVRWSet{
			Reads:  []*kvrwset.KVRead{{Key: "key1", Version: &kvrwset.Version{BlockNum: 1, TxNum: 1}}},
			Writes: kvWrite,
		}}}

	txRWSetBytes, err := txRwSet.ToProtoBytes()
	if err != nil {
		return nil, err
	}

	ccAction.Results = txRWSetBytes
	ccActionBytes, err := proto.Marshal(ccAction)
	if err != nil {
		return nil, err
	}
	prp.Extension = ccActionBytes
	prpBytes, err := proto.Marshal(prp)
	if err != nil {
		return nil, err
	}
	return prpBytes, nil
}

// SignECDSA sign with ec key
func SignECDSA(k *ecdsa.PrivateKey, digest []byte) (signature []byte, err error) {
	hash := sha256.New()
	_, err = hash.Write(digest)
	if err != nil {
		return nil, err
	}
	r, s, err := ecdsa.Sign(rand.Reader, k, hash.Sum(nil))
	if err != nil {
		return nil, err
	}

	s, _, err = utils.ToLowS(&k.PublicKey, s)
	if err != nil {
		return nil, err
	}

	return utils.MarshalECDSASignature(r, s)
}

// CertPem certificate
var CertPem = `-----BEGIN CERTIFICATE-----
MIICCjCCAbGgAwIBAgIQOcq9Om9VwUe9hGN0TTGw1DAKBggqhkjOPQQDAjBYMQsw
CQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMNU2FuIEZy
YW5jaXNjbzENMAsGA1UEChMET3JnMTENMAsGA1UEAxMET3JnMTAeFw0xNzA1MDgw
OTMwMzRaFw0yNzA1MDYwOTMwMzRaMGUxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpD
YWxpZm9ybmlhMRYwFAYDVQQHEw1TYW4gRnJhbmNpc2NvMRUwEwYDVQQKEwxPcmcx
LXNlcnZlcjExEjAQBgNVBAMTCWxvY2FsaG9zdDBZMBMGByqGSM49AgEGCCqGSM49
AwEHA0IABAm+2CZhbmsnA+HKQynXKz7fVZvvwlv/DdNg3Mdg7lIcP2z0b07/eAZ5
0chdJNcjNAd/QAj/mmGG4dObeo4oTKGjUDBOMA4GA1UdDwEB/wQEAwIFoDAdBgNV
HSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwDAYDVR0TAQH/BAIwADAPBgNVHSME
CDAGgAQBAgMEMAoGCCqGSM49BAMCA0cAMEQCIG55RvN4Boa0WS9UcIb/tI2YrAT8
EZd/oNnZYlbxxyvdAiB6sU9xAn4oYIW9xtrrOISv3YRg8rkCEATsagQfH8SiLg==
-----END CERTIFICATE-----`

// KeyPem ec private key
var KeyPem = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEICfXQtVmdQAlp/l9umWJqCXNTDurmciDNmGHPpxHwUK/oAoGCCqGSM49
AwEHoUQDQgAECb7YJmFuaycD4cpDKdcrPt9Vm+/CW/8N02Dcx2DuUhw/bPRvTv94
BnnRyF0k1yM0B39ACP+aYYbh05t6jihMoQ==
-----END EC PRIVATE KEY-----`)

// RootCA ca
var RootCA = `-----BEGIN CERTIFICATE-----
MIIB8TCCAZegAwIBAgIQU59imQ+xl+FmwuiFyUgFezAKBggqhkjOPQQDAjBYMQsw
CQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMNU2FuIEZy
YW5jaXNjbzENMAsGA1UEChMET3JnMTENMAsGA1UEAxMET3JnMTAeFw0xNzA1MDgw
OTMwMzRaFw0yNzA1MDYwOTMwMzRaMFgxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpD
YWxpZm9ybmlhMRYwFAYDVQQHEw1TYW4gRnJhbmNpc2NvMQ0wCwYDVQQKEwRPcmcx
MQ0wCwYDVQQDEwRPcmcxMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEFkpP6EqE
87ghFi25UWLvgPatxDiYKYaVSPvpo/XDJ0+9uUmK/C2r5Bvvxx1t8eTROwN77tEK
r+jbJIxX3ZYQMKNDMEEwDgYDVR0PAQH/BAQDAgGmMA8GA1UdJQQIMAYGBFUdJQAw
DwYDVR0TAQH/BAUwAwEB/zANBgNVHQ4EBgQEAQIDBDAKBggqhkjOPQQDAgNIADBF
AiEA1Xkrpq+wrmfVVuY12dJfMQlSx+v0Q3cYce9BE1i2mioCIAzqyduK/lHPI81b
nWiU9JF9dRQ69dEV9dxd/gzamfFU
-----END CERTIFICATE-----`
