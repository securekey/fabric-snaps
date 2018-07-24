/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package errors

//ErrorCode error code
type ErrorCode string

const (

	// PanicError will wrap snap panic into an error. Snap will try to handle panic by returning "panic-error" code to the caller.
	// The cause of panic will be logged as error in the snap log.
	PanicError = "panic-error"

	// SystemError ...
	SystemError = "system-error"

	// GeneralError ...
	GeneralError ErrorCode = "general-error"

	// MissingRequiredParameterError is returned when the caller does not provide required parameters for the call
	MissingRequiredParameterError = "missing-required-parameter"

	// ValidationError is usually caused by 'bad' data that is provided by the caller (e.g. invalid URL format)
	ValidationError ErrorCode = "validation-error"

	// InitializeConfigError ...
	InitializeConfigError = "initialize-config-error"

	// InitializeLoggingError ...
	InitializeLoggingError = "initialize-logging-error"

	// MissingConfigDataError ...
	MissingConfigDataError ErrorCode = "missing-config-data-error"

	// CryptoConfigError ...
	CryptoConfigError ErrorCode = "crypto-config-error"

	// CryptoError ...
	CryptoError ErrorCode = "crypto-error"

	// ParseCertError ...
	ParseCertError ErrorCode = "parse-cert-error"

	// ImportKeyError ...
	ImportKeyError ErrorCode = "import-key-error"

	// DecodePemError ...
	DecodePemError ErrorCode = "decode-pem-error"

	// GetKeyError ...
	GetKeyError ErrorCode = "get-key-error"

	// UnmarshalError ...
	UnmarshalError ErrorCode = "unmarshal-error"

	// InvalidFunctionError ...
	InvalidFunctionError ErrorCode = "invalid-function-error"

	// InitializeSnapError ...
	InitializeSnapError ErrorCode = "initialize-snap-error"

	// *** Start HTTP Snap *** //

	// HTTPClientError ...
	HTTPClientError ErrorCode = "http-client-error"

	// InvalidCertPinError ...
	InvalidCertPinError ErrorCode = "invalid-cert-pin-error"

	// *** End HTTP Snap *** //

	// InitMembershipServiceError ...
	InitMembershipServiceError = "init-membership-service-error"

	// MembershipError ... (errors from getAllPeers and getPeersOfChannel)
	MembershipError = "membership-error"

	// PeerConfigError ...
	PeerConfigError = "peer-config-error"

	// MissingChannelConfigError ...
	MissingChannelConfigError = "missing-channel-config-error"

	// ACLCheckError ...
	ACLCheckError = "acl-check-error"

	// *** Start Configuration Snap *** //

	// GetConfigError ...
	GetConfigError = "get-config-error"

	// *** End Configuration Snap *** //

	// *** Start Config Manager ***//

	// InvalidConfigMessage ...
	InvalidConfigMessage = "invalid-config-message"

	// InvalidPeerConfig ...
	InvalidPeerConfig = "invalid-peer-config"

	// InvalidAppConfig ...
	InvalidAppConfig = "invalid-app-config"

	// InvalidComponentConfig ....
	InvalidComponentConfig = "invalid-component-config"

	// InvalidConfigKey ...
	InvalidConfigKey = "invalid-config-key"
)
