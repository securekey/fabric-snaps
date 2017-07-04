/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"path/filepath"
	"testing"
)

func TestConfigurationPaths(t *testing.T) {
	err := Init("")
	if err != nil {
		t.Fatalf("Error initializing config: %s", err)
	}
	tlsCertPath := GetTLSCertPath()
	if !filepath.IsAbs(tlsCertPath) {
		t.Fatal("Expected absolute TLS filepath")
	}

}
