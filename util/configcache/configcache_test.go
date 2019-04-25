/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package configcache

import (
	"os"
	"testing"
)

func TestConfigCache(t *testing.T) {
	cache := New("core", "coreprefix", "./testdata")

	config, err := cache.Get("")
	if err != nil {
		t.Fatalf("error getting config: %s", err)
	}
	if config == nil {
		t.Fatalf("config is nil")
	}

	// Get again
	config, err = cache.Get("./testdata")
	if err != nil {
		t.Fatalf("error getting config: %s", err)
	}
	if config == nil {
		t.Fatalf("config is nil")
	}

	if _, err := cache.Get("./invalidpath"); err == nil {
		t.Fatalf("expecting error getting config for invalid path but got none")
	}
}

func TestFabricConfigPath(t *testing.T) {
	// set fabric config path
	os.Setenv("FABRIC_CFG_PATH", "./testfabricconfig")

	cache := New("core", "coreprefix", "./testdata")

	config, err := cache.Get("")
	if err != nil {
		t.Fatalf("error getting config: %s", err)
	}
	if config == nil {
		t.Fatalf("config is nil")
	}

	peerid := config.Get("peer.id")
	if peerid != "ckent" {
		t.Fatalf("incorrect config loaded")
	}

}
