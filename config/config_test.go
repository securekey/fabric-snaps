/*
   Copyright SecureKey Technologies Inc.
   This file contains software code that is the intellectual property of SecureKey.
   SecureKey reserves all rights in the code and you may not use it without written permission from SecureKey.
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