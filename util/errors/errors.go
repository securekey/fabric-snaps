/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package errors

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/xid"
)

// Error extends error with
// additional context
type Error interface {
	error

	// Status returns the error code
	ErrorCode() ErrorCode
	// Generate log msg
	GenerateLogMsg() string
	// Generate client error msg
	GenerateClientErrorMsg() string
}

type customError struct {
	error
	code    ErrorCode
	errorID string
}

// New returns a new Error
func New(code ErrorCode, msg string) Error {
	return &customError{
		error:   errors.New(msg),
		code:    code,
		errorID: generateErrorID(),
	}
}

// Wrap returns a new Error
func Wrap(code ErrorCode, cause error, msg string) Error {
	return &customError{
		error:   errors.Wrap(cause, msg),
		code:    code,
		errorID: generateErrorID(),
	}
}

// Wrapf returns a new Error
func Wrapf(code ErrorCode, cause error, fmt string, args ...interface{}) Error {
	return &customError{
		error:   errors.Wrapf(cause, fmt, args...),
		code:    code,
		errorID: generateErrorID(),
	}
}

// Errorf returns a new Error
func Errorf(code ErrorCode, fmt string, args ...interface{}) Error {
	return &customError{
		error:   errors.Errorf(fmt, args...),
		code:    code,
		errorID: generateErrorID(),
	}
}

// CreateError returns custom error
func CreateError(err error, code ErrorCode, msg string) Error {
	errorObj, ok := GetError(err)
	if !ok {
		errorObj = WithMessage(code, err, msg)
	}
	return errorObj
}

// GetError returns custom error
func GetError(err error) (Error, bool) {
	if s, ok := err.(Error); ok {
		return s, true
	}
	unwrappedErr := errors.Cause(err)
	if s, ok := unwrappedErr.(Error); ok {
		return s, true
	}

	return nil, false
}

// WithMessage returns a new Error
func WithMessage(code ErrorCode, err error, msg string) Error {
	return &customError{
		error:   errors.WithMessage(err, msg),
		code:    code,
		errorID: generateErrorID(),
	}
}

// generateErrorID return error ID
func generateErrorID() string {
	return xid.New().String()
}

// ErrorCode returns the error code
func (e *customError) ErrorCode() ErrorCode {
	return e.code
}

// ErrorID returns the error ID
func (e *customError) ErrorID() string {
	return e.errorID
}

// GenerateLogMsg returns the log msg
func (e *customError) GenerateLogMsg() string {
	return fmt.Sprintf("errorID:%s errorCode:%s error:%v", e.errorID, e.code, e.error)
}

// GenerateClientErrorMsg returns the client error msg
func (e *customError) GenerateClientErrorMsg() string {
	return fmt.Sprintf("errorID:%s errorCode:%s error:%v", e.errorID, e.code, e.error)
}
