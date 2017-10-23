/*
Copyright IBM Corp. 2017 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
/*
Notice: This file has been modified for Hyperledger Fabric SDK Go usage.
Please review third_party pinning scripts and patches for more details.
*/

package ccprovider

import (
	"context"

	"github.com/golang/protobuf/proto"
	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/internal/github.com/hyperledger/fabric/core/ledger"
	flogging "github.com/securekey/fabric-snaps/internal/github.com/hyperledger/fabric/sdkpatch/logbridge"
)

var ccproviderLogger = flogging.MustGetLogger("ccprovider")

var chaincodeInstallPath string

//CCPackage encapsulates a chaincode package which can be
//    raw ChaincodeDeploymentSpec
//    SignedChaincodeDeploymentSpec
// Attempt to keep the interface at a level with minimal
// interface for possible generalization.
type CCPackage interface {
	//InitFromBuffer initialize the package from bytes
	InitFromBuffer(buf []byte) (*ChaincodeData, error)

	// InitFromFS gets the chaincode from the filesystem (includes the raw bytes too)
	InitFromFS(ccname string, ccversion string) ([]byte, *pb.ChaincodeDeploymentSpec, error)

	// PutChaincodeToFS writes the chaincode to the filesystem
	PutChaincodeToFS() error

	// GetDepSpec gets the ChaincodeDeploymentSpec from the package
	GetDepSpec() *pb.ChaincodeDeploymentSpec

	// GetDepSpecBytes gets the serialized ChaincodeDeploymentSpec from the package
	GetDepSpecBytes() []byte

	// ValidateCC validates and returns the chaincode deployment spec corresponding to
	// ChaincodeData. The validation is based on the metadata from ChaincodeData
	// One use of this method is to validate the chaincode before launching
	ValidateCC(ccdata *ChaincodeData) error

	// GetPackageObject gets the object as a proto.Message
	GetPackageObject() proto.Message

	// GetChaincodeData gets the ChaincodeData
	GetChaincodeData() *ChaincodeData

	// GetId gets the fingerprint of the chaincode based on package computation
	GetId() []byte
}

type CCCacheSupport interface {
	//GetChaincode is needed by the cache to get chaincode data
	GetChaincode(ccname string, ccversion string) (CCPackage, error)
}

// CCInfoFSImpl provides the implementation for CC on the FS and the access to it
// It implements CCCacheSupport
type CCInfoFSImpl struct{}

// The following lines create the cache of CCPackage data that sits
// on top of the file system and avoids a trip to the file system
// every time. The cache is disabled by default and only enabled
// if EnableCCInfoCache is called. This is an unfortunate hack
// required by some legacy tests that remove chaincode packages
// from the file system as a means of simulating particular test
// conditions. This way of testing is incompatible with the
// immutable nature of chaincode packages that is assumed by hlf v1
// and implemented by this cache. For this reason, tests are for now
// allowed to run with the cache disabled (unless they enable it)
// until a later time in which they are fixed. The peer process on
// the other hand requires the benefits of this cache and therefore
// enables it.
// TODO: (post v1) enable cache by default as soon as https://jira.hyperledger.org/browse/FAB-3785 is completed

// ccInfoFSStorageMgr is the storage manager used either by the cache or if the
// cache is bypassed
var ccInfoFSProvider = &CCInfoFSImpl{}

// ccInfoCache is the cache instance itself

// ccInfoCacheEnabled keeps track of whether the cache is enable
// (it is disabled by default)
var ccInfoCacheEnabled bool

//CCContext pass this around instead of string of args
type CCContext struct {
	//ChainID chain id
	ChainID string

	//Name chaincode name
	Name string

	//Version used to construct the chaincode image and register
	Version string

	//TxID is the transaction id for the proposal (if any)
	TxID string

	//Syscc is this a system chaincode
	Syscc bool

	//SignedProposal for this invoke (if any)
	//this is kept here for access control and in case we need to pass something
	//from this to the chaincode
	SignedProposal *pb.SignedProposal

	//Proposal for this invoke (if any)
	//this is kept here just in case we need to pass something
	//from this to the chaincode
	Proposal *pb.Proposal

	//this is not set but computed (note that this is not exported. use GetCanonicalName)
	canonicalName string

	// this is additional data passed to the chaincode
	ProposalDecorations map[string][]byte
}

//-------- ChaincodeData is stored on the LSCC -------

//ChaincodeData defines the datastructure for chaincodes to be serialized by proto
//Type provides an additional check by directing to use a specific package after instantiation
//Data is Type specifc (see CDSPackage and SignedCDSPackage)
type ChaincodeData struct {
	//Name of the chaincode
	Name string `protobuf:"bytes,1,opt,name=name"`

	//Version of the chaincode
	Version string `protobuf:"bytes,2,opt,name=version"`

	//Escc for the chaincode instance
	Escc string `protobuf:"bytes,3,opt,name=escc"`

	//Vscc for the chaincode instance
	Vscc string `protobuf:"bytes,4,opt,name=vscc"`

	//Policy endorsement policy for the chaincode instance
	Policy []byte `protobuf:"bytes,5,opt,name=policy,proto3"`

	//Data data specific to the package
	Data []byte `protobuf:"bytes,6,opt,name=data,proto3"`

	//Id of the chaincode that's the unique fingerprint for the CC
	//This is not currently used anywhere but serves as a good
	//eyecatcher
	Id []byte `protobuf:"bytes,7,opt,name=id,proto3"`

	//InstantiationPolicy for the chaincode
	InstantiationPolicy []byte `protobuf:"bytes,8,opt,name=instantiation_policy,proto3"`
}

//implement functions needed from proto.Message for proto's mar/unmarshal functions

//Reset resets
func (cd *ChaincodeData) Reset() { *cd = ChaincodeData{} }

//String converts to string
func (cd *ChaincodeData) String() string { return proto.CompactTextString(cd) }

//ProtoMessage just exists to make proto happy
func (*ChaincodeData) ProtoMessage() {}

// ChaincodeProvider provides an abstraction layer that is
// used for different packages to interact with code in the
// chaincode package without importing it; more methods
// should be added below if necessary
type ChaincodeProvider interface {
	// GetContext returns a ledger context
	GetContext(ledger ledger.PeerLedger, txid string) (context.Context, error)
	// GetCCContext returns an opaque chaincode context
	GetCCContext(cid, name, version, txid string, syscc bool, signedProp *pb.SignedProposal, prop *pb.Proposal) interface{}
	// GetCCValidationInfoFromLSCC returns the VSCC and the policy listed by LSCC for the supplied chaincode
	GetCCValidationInfoFromLSCC(ctxt context.Context, txid string, signedProp *pb.SignedProposal, prop *pb.Proposal, chainID string, chaincodeID string) (string, []byte, error)
	// ExecuteChaincode executes the chaincode given context and args
	ExecuteChaincode(ctxt context.Context, cccid interface{}, args [][]byte) (*pb.Response, *pb.ChaincodeEvent, error)
	// Execute executes the chaincode given context and spec (invocation or deploy)
	Execute(ctxt context.Context, cccid interface{}, spec interface{}) (*pb.Response, *pb.ChaincodeEvent, error)
	// ExecuteWithErrorFilter executes the chaincode given context and spec and returns payload
	ExecuteWithErrorFilter(ctxt context.Context, cccid interface{}, spec interface{}) ([]byte, *pb.ChaincodeEvent, error)
	// Stop stops the chaincode given context and deployment spec
	Stop(ctxt context.Context, cccid interface{}, spec *pb.ChaincodeDeploymentSpec) error
	// ReleaseContext releases the context returned previously by GetContext
	ReleaseContext()
}

var ccFactory ChaincodeProviderFactory

// ChaincodeProviderFactory defines a factory interface so
// that the actual implementation can be injected
type ChaincodeProviderFactory interface {
	NewChaincodeProvider() ChaincodeProvider
}
