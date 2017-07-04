/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package registry

import (
	"testing"

	"github.com/securekey/fabric-snaps/api/config"
	"github.com/securekey/fabric-snaps/pkg/snaps/examplesnap"
)

func TestInvalidSnapNameSnap(t *testing.T) {
	snapName := "somesnap"

	registry := NewSnapsRegistry(nil)

	err := registry.Initialize()
	if err != nil {
		t.Fatalf(err.Error())
	}

	snap := registry.GetSnap(snapName)
	if snap != nil {
		t.Fatalf("Expecting nil but got non nil snap")
	}
}

func TestLocalSnap(t *testing.T) {
	snapName := "examplesnap"

	var snaps []*config.SnapConfig
	snaps = append(snaps, &config.SnapConfig{
		Name: snapName,
		Snap: &examplesnap.ExampleSnap{},
	})

	registry := NewSnapsRegistry(snaps)

	err := registry.Initialize()
	if err != nil {
		t.Fatalf(err.Error())
	}

	snap := registry.GetSnap(snapName)
	if snap == nil {
		t.Fatalf("Expecting snap but got nil")
	}
	if snap.Name != snapName {
		t.Fatalf("Expecting snap.Name to be %s but got %s", snapName, snap.Name)
	}
	if !snap.Enabled {
		t.Fatalf("Expecting snap.Enabled to be true but got false")
	}
	if snap.Snap == nil {
		t.Fatalf("Expecting snap.Snap to be non nil but got nil")
	}
	if snap.SnapURL != "" {
		t.Fatalf("Expecting snap.SnapURL to be '' but got %s", snap.SnapURL)
	}

	if len(snap.InitArgsStr) != 2 {
		t.Fatalf("Expecting 2 init args but got %d", len(snap.InitArgs))
	}
	expectedArg1 := "example snap init arg"
	expectedArg2 := "second argument"
	if snap.InitArgsStr[0] != expectedArg1 {
		t.Fatalf("Expecting first init arg to be %s but got %s", expectedArg1, snap.InitArgs[0])
	}
	if snap.InitArgsStr[1] != expectedArg2 {
		t.Fatalf("Expecting second init arg to be %s but got %s", expectedArg2, snap.InitArgs[1])
	}
}

func TestRemoteSnap(t *testing.T) {
	snapName := "exampleremotesnap"

	var snaps []*config.SnapConfig
	snaps = append(snaps, &config.SnapConfig{
		Name: snapName,
	})

	registry := NewSnapsRegistry(snaps)

	err := registry.Initialize()
	if err != nil {
		t.Fatalf(err.Error())
	}

	snap := registry.GetSnap(snapName)
	if snap == nil {
		t.Fatalf("Expecting snap but got nil")
	}
	if snap.Name != snapName {
		t.Fatalf("Expecting snap.Name to be %s but got %s", snapName, snap.Name)
	}
	if !snap.Enabled {
		t.Fatalf("Expecting snap.Enabled to be true but got false")
	}
	if snap.Snap == nil {
		t.Fatalf("Expecting snap.Snap to be non nil but got nil")
	}
	expectedSnapURL := "remote.url.address:8009"
	if snap.SnapURL != expectedSnapURL {
		t.Fatalf("Expecting snap.SnapURL to be %s but got %s", expectedSnapURL, snap.SnapURL)
	}
	if !snap.TLSEnabled {
		t.Fatalf("Expecting snap.TLSEnabled to be true but got false")
	}
	expectedTLSRootCert := "/etc/hyperledger/msp/snaps/remote.url.address/cacerts/ca-cert.pem"
	if snap.TLSRootCertFile != expectedTLSRootCert {
		t.Fatalf("Expecting snap.TLSRootCertFile to be %s but got %s", expectedTLSRootCert, snap.TLSRootCertFile)
	}
	if len(snap.InitArgsStr) != 0 {
		t.Fatalf("Expecting 0 init args but got %d", len(snap.InitArgs))
	}
}
