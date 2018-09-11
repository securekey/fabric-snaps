/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package initbcinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/securekey/fabric-snaps/mocks/mockbcinfo"
)

func TestInitBCInfo(t *testing.T) {
	channelID := "testchannel"
	initialBlockHeight := uint64(1002)

	bcInfo, ok := Get(channelID)
	require.Falsef(t, ok, "expecting blockchain info not to be set yet")

	err := Set(channelID, mockbcinfo.BCInfo(initialBlockHeight))
	require.NoError(t, err)
	err = Set(channelID, mockbcinfo.BCInfo(initialBlockHeight))
	require.Errorf(t, err, "expecting error setting initial blockchain info twice")

	bcInfo, ok = Get(channelID)
	require.True(t, ok)
	require.NotNil(t, bcInfo)
	assert.Equal(t, initialBlockHeight, bcInfo.Height)
}
