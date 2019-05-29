/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mspid

import (
	"testing"

	"github.com/stretchr/testify/require"

	membershipMocks "github.com/securekey/fabric-snaps/membershipsnap/pkg/mocks"
)

const (
	org1MSP = "Org1MSP"
)

func TestPeerFilter(t *testing.T) {
	_, err := New([]string{})
	if err == nil {
		t.Fatal("Expecting error when no channel ID provided but got none")
	}

	mspID := org1MSP

	f, err := New([]string{mspID})
	require.NoErrorf(t, err, "Got error when creating peer filter")

	require.False(t, f.Accept(membershipMocks.New("p1", "msp", 0)))

	require.True(t, f.Accept(membershipMocks.New("p2", org1MSP, 0)))
}
