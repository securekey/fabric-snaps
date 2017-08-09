/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package binary

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"strings"

	cutil "github.com/hyperledger/fabric/core/container/util"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// Platform for chaincodes written in Go
type Platform struct {
}

// ValidateSpec validates binary chaincodes
func (binaryPlatform *Platform) ValidateSpec(spec *pb.ChaincodeSpec) error {

	// Nothing to validate as of now for binary chaincodes
	return nil
}

func (binaryPlatform *Platform) ValidateDeploymentSpec(cds *pb.ChaincodeDeploymentSpec) error {

	if cds.CodePackage == nil || len(cds.CodePackage) == 0 {
		// Nothing to validate if no CodePackage was included
		return nil
	}

	//We do not want to allow something like ./pkg/shady.a to be installed under
	// $GOPATH within the container.
	//
	// It should be noted that we cannot catch every threat with these techniques.  Therefore,
	// the container itself needs to be the last line of defense and be configured to be
	// resilient in enforcing constraints. However, we should still do our best to keep as much
	// garbage out of the system as possible.
	is := bytes.NewReader(cds.CodePackage)
	gr, err := gzip.NewReader(is)
	if err != nil {
		return fmt.Errorf("failure opening codepackage gzip stream: %s", err)
	}
	tr := tar.NewReader(gr)

	header, err := tr.Next()
	if err != nil {
		//It means tar is empty
		return fmt.Errorf("No entries found inside codepackage gzip %s", err)
	}

	if header.Name != "chaincode" {
		return fmt.Errorf("illegal file name detected for file %s", header.Name)
	}
	// --------------------------------------------------------------------------------------
	// Check that file mode makes sense
	// --------------------------------------------------------------------------------------
	// Acceptable flags:
	//      ISREG      == 0100000
	//      rwxrwxrwx == 0555
	//
	// Anything else is suspect in this context and will be rejected
	// --------------------------------------------------------------------------------------
	if header.Mode&^0100555 != 0 {
		return fmt.Errorf("illegal file mode detected for file %s: %o", header.Name, header.Mode)
	}

	header, err = tr.Next()
	if err == nil {
		return fmt.Errorf("Muliple entries found inside code package gzip ")
	}

	return nil
}

// Generates a deployment payload for GOLANG as a series of src/$pkg entries in .tar.gz format
func (binaryPlatform *Platform) GetDeploymentPayload(spec *pb.ChaincodeSpec) ([]byte, error) {

	var err error

	// --------------------------------------------------------------------------------------
	// Write out binary to our tar package
	// --------------------------------------------------------------------------------------
	payload := bytes.NewBuffer(nil)
	gw := gzip.NewWriter(payload)
	tw := tar.NewWriter(gw)

	err = cutil.WriteFileToPackage(spec.ChaincodeId.Path, "chaincode", tw, 0100555)
	if err != nil {
		return nil, fmt.Errorf("Error writing %s to tar: %s", spec.ChaincodeId.Path, err)
	}

	tw.Close()
	gw.Close()

	return payload.Bytes(), nil
}

func (binaryPlatform *Platform) GenerateDockerfile(cds *pb.ChaincodeDeploymentSpec) (string, error) {

	var buf []string

	buf = append(buf, "FROM "+cutil.GetDockerfileFromConfig("chaincode.binary.runtime"))
	buf = append(buf, "ADD binpackage.tar /usr/local/bin")

	dockerFileContents := strings.Join(buf, "\n")

	return dockerFileContents, nil
}

func (binaryPlatform *Platform) GenerateDockerBuild(cds *pb.ChaincodeDeploymentSpec, tw *tar.Writer) error {

	return cutil.WriteBytesToPackage("binpackage.tar", cds.CodePackage, tw)
}
