/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"fmt"

	"encoding/json"

	"github.com/DATA-DOG/godog"
	configapi "github.com/securekey/fabric-snaps/configurationsnap/api"
)

//ConfigurationSnapSteps Configuration Snap BDD test steps
type ConfigurationSnapSteps struct {
	BDDContext         *BDDContext
	LastInvokeResponse [][]byte
}

/*
 * Channel query methods used by this test
 */
const (
	getPublicKeyForLogging string = "getPublicKeyForLogging"
)

// NewConfigurationSnapSteps ...
func NewConfigurationSnapSteps(context *BDDContext) *ConfigurationSnapSteps {
	return &ConfigurationSnapSteps{BDDContext: context}
}

//verifyPublicKeyForLoggingResults checks for nil error, not nil response and response matching expected params passe; if not then throw error
func (d *ConfigurationSnapSteps) verifyPublicKeyForLoggingResults(publickey string, keyID string) error {

	response := &configapi.PublicKeyForLogging{}
	json.Unmarshal(d.LastInvokeResponse[0], response)

	if response.PublicKey != publickey || response.KeyID != keyID {
		return fmt.Errorf("unexpected publickey, keyid combination found. Expected '{%s, %s}' but got '{%s, %s}'", publickey, keyID, response.PublicKey, response.KeyID)
	}

	return nil
}

func (d *ConfigurationSnapSteps) registerSteps(s *godog.Suite) {
	s.BeforeScenario(d.BDDContext.beforeScenario)
	s.AfterScenario(d.BDDContext.afterScenario)
	s.Step(`^client C1 receives public key "([^"]*)" and key id "([^"]*)"$`, d.verifyPublicKeyForLoggingResults)
}
