/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"testing"
	"github.com/securekey/fabric-snaps/config"
)

func TestConfigInit(t *testing.T) {
	err := config.Init("")
	if err != nil {
		t.Fatalf("Error initializing config from daemon: %s", err)
	}
}