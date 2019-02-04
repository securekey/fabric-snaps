/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package errors

import (
	"testing"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestCause(t *testing.T) {
	rootCause := status.New(status.ChaincodeStatus, 412, "412 error occurred", nil)
	err1 := WithMessage(GeneralError, rootCause, "some error")

	cause := errors.Cause(err1)
	assert.NotNil(t, cause)
	assert.Equal(t, rootCause, cause)

	err2 := WithMessage(SystemError, err1, "some other error")
	cause = errors.Cause(err2)
	assert.NotNil(t, cause)
	assert.Equal(t, rootCause, cause)

	stat, ok := status.FromError(err1)
	assert.True(t, ok)
	assert.Equal(t, rootCause, stat)

	stat, ok = status.FromError(err2)
	assert.True(t, ok)
	assert.Equal(t, rootCause, stat)
}
