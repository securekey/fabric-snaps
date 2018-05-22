/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"fmt"

	"strings"

	"github.com/DATA-DOG/godog"
)

var queryResult string

// ConfigCLISteps config cli BDD test steps
type ConfigCLISteps struct {
	BDDContext *BDDContext
}

// NewConfigCLISteps new config cli steps
func NewConfigCLISteps(context *BDDContext) *ConfigCLISteps {
	return &ConfigCLISteps{BDDContext: context}
}

// updateConfig update config using config cli
func (c *ConfigCLISteps) updateConfig(configFile, mspID, channelID string) error {
	_, err := c.BDDContext.configCLI.ExecUpdate(channelID, mspID, configFile)
	if err != nil {
		return fmt.Errorf("failed to update config: %v", err)
	}
	return nil
}

// exec executes action on config using config cli
func (c *ConfigCLISteps) exec(action, peerID, mspID, appName, version, channelID string) error {
	var err error
	queryResult, err = c.BDDContext.configCLI.Exec(action, channelID, mspID, peerID, appName, version)
	if err != nil {
		return fmt.Errorf("failed to %s config: %v", action, err)
	}
	return nil
}

func (c *ConfigCLISteps) containsInQueryResult(value string) error {
	if queryResult == "" {
		return fmt.Errorf("queryResult is empty")
	}
	logger.Infof("Query value %s and tested value %s", queryResult, value)
	if !strings.Contains(queryResult, value) {
		return fmt.Errorf("query value(%s) doesn't contain expected value(%s)", queryResult, value)
	}
	return nil
}

func (c *ConfigCLISteps) notContainsInQueryResult(value string) error {
	logger.Infof("Query value %s and tested value %s", queryResult, value)
	if strings.Contains(queryResult, value) {
		return fmt.Errorf("query value(%s) shoud not contain expected value(%s)", queryResult, value)
	}
	return nil
}

func (c *ConfigCLISteps) registerSteps(s *godog.Suite) {
	s.BeforeScenario(c.BDDContext.BeforeScenario)
	s.AfterScenario(c.BDDContext.AfterScenario)
	s.Step(`^client update config "([^"]*)" with mspid "([^"]*)" on the "([^"]*)" channel$`, c.updateConfig)
	s.Step(`^client "([^"]*)" config by peer id "([^"]*)" with mspid "([^"]*)" with app name "([^"]*)" with version "([^"]*)" on the "([^"]*)" channel$`, c.exec)
	s.Step(`^response from cli query to client contains value "([^"]*)"$`, c.containsInQueryResult)
	s.Step(`^response from cli query to client not contains value "([^"]*)"$`, c.notContainsInQueryResult)

}
