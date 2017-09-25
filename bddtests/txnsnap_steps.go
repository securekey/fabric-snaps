/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"math/rand"

	"github.com/DATA-DOG/godog"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// TxnSnapSteps ...
type TxnSnapSteps struct {
	BDDContext *BDDContext
}

// NewTxnSnapSteps ...
func NewTxnSnapSteps(context *BDDContext) *TxnSnapSteps {
	return &TxnSnapSteps{BDDContext: context}
}

func (t *TxnSnapSteps) queryWithLargePayload(ccID, channelID string) error {
	args := "txnsnap,endorseTransaction,mychannel,example_cc3,invoke,delete," + getLargeString(200000)

	common := NewCommonSteps(t.BDDContext)
	return common.queryCC(ccID, channelID, args)
}

func getLargeString(size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (t *TxnSnapSteps) registerSteps(s *godog.Suite) {
	s.BeforeScenario(t.BDDContext.beforeScenario)
	s.AfterScenario(t.BDDContext.afterScenario)
	s.Step(`^client C1 query chaincode "([^"]*)" on channel "([^"]*)" with a large payload on p0 and succeeds$`, t.queryWithLargePayload)
}
