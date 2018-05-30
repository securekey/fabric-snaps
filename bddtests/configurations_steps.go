/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	bccsputils "github.com/hyperledger/fabric/bccsp/utils"

	"github.com/DATA-DOG/godog"
	"github.com/pkg/errors"
)

// ConfigurationsSnapSteps ...
type ConfigurationsSnapSteps struct {
	BDDContext *BDDContext
}

// NewConfigurationsSnapSteps ...
func NewConfigurationsSnapSteps(context *BDDContext) *ConfigurationsSnapSteps {
	return &ConfigurationsSnapSteps{BDDContext: context}
}

//checkCSR to verify that CSR was created
func (c *ConfigurationsSnapSteps) checkCSR(ccID string) error {
	//key bytes returned
	if queryValue == "" {
		return fmt.Errorf("QueryValue is empty")
	}
	if strings.Contains(queryValue, "Error") {
		return fmt.Errorf("QueryValue contains error: %s", queryValue)
	}
	//response contains public key bytes
	raw := []byte(queryValue)
	csr := pem.EncodeToMemory(&pem.Block{
		Type: "CERTIFICATE REQUEST", Bytes: raw,
	})

	logger.Debugf("CSR was created [%v]", string(csr))
	//returned certificate request should have fields configured in config.yaml
	cr, e := x509.ParseCertificateRequest(raw)
	if e != nil {
		return e
	}
	if cr.Subject.Organization[0] == "" {
		return errors.Errorf("CSR should have non nil subject-organization")
	}
	logger.Debugf("CSR was created [%v]", cr.Subject)
	return nil
}

func (c *ConfigurationsSnapSteps) checkKeyGenResponse(ccID string, expectedKeyType string) error {
	//key bytes returned
	if queryValue == "" {
		return fmt.Errorf("QueryValue is empty")
	}
	if strings.Contains(queryValue, "Error") {
		return fmt.Errorf("QueryValue contains error: %s", queryValue)
	}
	//response contains public key bytes
	raw := []byte(queryValue)
	pk, err := bccsputils.DERToPublicKey(raw)
	if err != nil {
		return errors.Wrap(err, "failed marshalling der to public key")
	}
	switch k := pk.(type) {
	case *ecdsa.PublicKey:
		if !strings.Contains(expectedKeyType, "ECDSA") {
			return errors.Errorf("Expected ECDSA key but got [%v]", k)
		}
		ecdsaPK, ok := pk.(*ecdsa.PublicKey)
		if !ok {
			return errors.New("failed casting to ECDSA public key. Invalid raw material")
		}
		ecPt := elliptic.Marshal(ecdsaPK.Curve, ecdsaPK.X, ecdsaPK.Y)
		hash := sha256.Sum256(ecPt)
		ski := hash[:]
		if len(ski) == 0 {
			return errors.New("Expected valid SKI for PK")
		}

	case *rsa.PublicKey:
		if !strings.Contains(expectedKeyType, "RSA") {
			return errors.Errorf("Expected RSA key but got [%v]", k)
		}
		rsaPK, ok := pk.(*rsa.PublicKey)
		if !ok {
			return errors.New("failed casting to RSA public key. Invalid raw material")
		}
		PubASN1, err := x509.MarshalPKIXPublicKey(rsaPK)
		if err != nil {
			return err
		}
		if len(PubASN1) == 0 {
			return errors.New("Invalid RSA key")
		}
	default:
		logger.Debugf("Not supported type: '%s'", k)
		return errors.Errorf("Received unsupported key type")
	}

	return nil
}

func (c *ConfigurationsSnapSteps) registerSteps(s *godog.Suite) {
	s.BeforeScenario(c.BDDContext.BeforeScenario)
	s.AfterScenario(c.BDDContext.AfterScenario)
	s.Step(`^response from "([^"]*)" to client C1 has key and key type is "([^"]*)" on p0$`, c.checkKeyGenResponse)
	s.Step(`^response from "([^"]*)" to client C1 has CSR on p0$`, c.checkCSR)
}
