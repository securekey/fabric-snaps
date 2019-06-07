// +build testing

/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txsnapservice

import (
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/validator"
)

// SetValidatorProvider sets the validator provider for unit tests
func SetValidatorProvider(provider func(channelID string) validator.Validator) {
	getValidator = provider
}
