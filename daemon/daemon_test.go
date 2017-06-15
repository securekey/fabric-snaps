/*
   Copyright SecureKey Technologies Inc.
   This file contains software code that is the intellectual property of SecureKey.
   SecureKey reserves all rights in the code and you may not use it without written permission from SecureKey.
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