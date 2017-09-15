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
	peerConfig, err := t.BDDContext.Client.Config().PeerConfig("peerorg1", "peer0")
	if err != nil {
		return fmt.Errorf("Error reading peer config: %s", err)
	}

	if queryValue == "" {
		return fmt.Errorf("QueryValue is empty")
	}
	if !strings.Contains(queryValue, peerConfig.Host) && !strings.Contains(queryValue, peerConfig.TLS.ServerHostOverride) {
		return fmt.Errorf("Query value(%s) doesn't contain expected value(%s) or value(%s)",
			queryValue, peerConfig.Host, peerConfig.TLS.ServerHostOverride)
	}
	return nil
}

func (t *TxnSnapSteps) registerSteps(s *godog.Suite) {
	s.BeforeScenario(t.BDDContext.beforeScenario)
	s.AfterScenario(t.BDDContext.afterScenario)
	s.Step(`^response from "([^"]*)" to client C1 contains value p0$`, t.assertMembershipResponse)
}
