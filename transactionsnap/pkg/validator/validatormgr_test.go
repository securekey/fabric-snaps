/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	channel1ID = "channel1"
	channel2ID = "channel2"
)

func TestValidatorMgr_ValidatorForChannel(t *testing.T) {
	v := Get().ValidatorForChannel(channel1ID)
	require.NotNil(t, v)

	v2 := Get().ValidatorForChannel(channel1ID)
	assert.Equal(t, v, v2)

	v3 := Get().ValidatorForChannel(channel2ID)
	assert.NotEqual(t, v, v3)
}
