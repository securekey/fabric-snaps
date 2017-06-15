/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"testing"

)

func TestConfigInit(t *testing.T) {
	err := Init("")
	if err != nil {
		t.Fatalf("Error initializing config: %s", err)
	}
}