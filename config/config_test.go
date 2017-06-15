/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestConfigInit(t *testing.T) {
	err := Init("")
	if err != nil {
		t.Fatalf("Error initializing config: %s", err)
	}
}

func TestGetConfigPath(t *testing.T) {
	err := Init("")
	if err != nil {
		t.Fatalf("Error initializing config: %s", err)
	}
	tlsCertPath := GetTLSCertPath()
	if !filepath.IsAbs(tlsCertPath) {
		t.Fatal("Expected absolute TLS filepath")
	}

	port := GetSnapServerPort()
	fmt.Println(port)

	isEnabled := IsTLSEnabled()
	fmt.Println(isEnabled)
	fmt.Println(GetTLSRootCertPath())
}
