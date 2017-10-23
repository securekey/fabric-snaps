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

// TxnSnapSteps ...
type TxnSnapSteps struct {
	BDDContext *BDDContext
}

// NewTxnSnapSteps ...
func NewTxnSnapSteps(context *BDDContext) *TxnSnapSteps {
	return &TxnSnapSteps{BDDContext: context}
}

func (t *TxnSnapSteps) assertMembershipResponse(arg string) error {
	peerConfig, err := t.BDDContext.Client.Config().PeerConfig("peerorg1", "peer0.org1.example.com")
	if err != nil {
		return fmt.Errorf("Error reading peer config: %s", err)
	}

	if queryValue == "" {
		return fmt.Errorf("QueryValue is empty")
	}

	serverHostOverride := ""
	if str, ok := peerConfig.GRPCOptions["ssl-target-name-override"].(string); ok {
		serverHostOverride = str
	}

	if !strings.Contains(queryValue, peerConfig.URL) && !strings.Contains(queryValue, serverHostOverride) {
		return fmt.Errorf("Query value(%s) doesn't contain expected value(%s) or value(%s)",
			queryValue, peerConfig.URL, serverHostOverride)
	}
	return nil
}

func (t *TxnSnapSteps) registerSteps(s *godog.Suite) {
	s.BeforeScenario(t.BDDContext.beforeScenario)
	s.AfterScenario(t.BDDContext.afterScenario)
	s.Step(`^response from "([^"]*)" to client C1 contains value p0$`, t.assertMembershipResponse)
}
