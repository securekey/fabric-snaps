/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package errors

//ErrorCode error code
type ErrorCode int

const (
	// GeneralError generic error
	GeneralError ErrorCode = 0

	// ValidationError ...
	ValidationError ErrorCode = 1

	// MissingConfigDataError ...
	MissingConfigDataError ErrorCode = 2

	// CryptoConfigError ...
	CryptoConfigError ErrorCode = 3

	// CryptoError ...
	CryptoError ErrorCode = 4

	// HTTPClientError ...
	HTTPClientError ErrorCode = 5

	// InvalidCertPinError ...
	InvalidCertPinError ErrorCode = 6

	// ParseCertError ...
	ParseCertError ErrorCode = 7

	// ImportKeyError ...
	ImportKeyError ErrorCode = 8

	// DecodePemError ...
	DecodePemError ErrorCode = 9

	// GetKeyError ...
	GetKeyError ErrorCode = 10

	// UnmarshallError ...
	UnmarshallError ErrorCode = 11

	// MissingRequiredParameterError ...
	MissingRequiredParameterError = 12

	// PanicError ...
	PanicError = 13

	// SystemError ...
	SystemError = 14
)
