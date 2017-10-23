/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
/*
Notice: This file has been modified for Hyperledger Fabric SDK Go usage.
Please review third_party pinning scripts and patches for more details.
*/

package logbridge

import (
	"io"

	"github.com/hyperledger/fabric-sdk-go/pkg/logging"
)

type Level int

// Log levels (from fabric-sdk-go/pkg/logging/level.go).
const (
	CRITICAL logging.Level = logging.CRITICAL
	ERROR    logging.Level = logging.ERROR
	WARNING  logging.Level = logging.WARNING
	INFO     logging.Level = logging.INFO
	DEBUG    logging.Level = logging.DEBUG
	NOTICE   logging.Level = logging.WARNING
)

// Logger bridges the SDK's logger struct
type Logger struct {
	*logging.Logger
	Module string
}

// MustGetLogger bridges calls the Go SDK NewLogger
func MustGetLogger(module string) *Logger {
	fabModule := "fabric_sdk_go"
	logger := logging.NewLogger(fabModule)
	return &Logger{
		Logger: logger,
		Module: fabModule,
	}
}

//DefaultLevel
func DefaultLevel() string {
	return logging.INFO.String()
}

// SetLevel
func SetLevel(level Level, module string) {
	logging.SetLevel(logging.Level(level), module)
}

// LogLevel returns the log level from a string representation.
func LogLevel(level string) (logging.Level, error) {
	return logging.LogLevel(level)
}

// Warningf bridges calls to the Go SDK logger's Warnf. .....
func (l *Logger) Warningf(format string, args ...interface{}) {
	l.Warnf(format, args...)
}

// Warning bridges calls to the Go SDK logger's Warn.
func (l *Logger) Warning(args ...interface{}) {
	l.Warn(args...)
}

// Noticef bridges calls to the Go SDK logger's Warnf. .....
func (l *Logger) Noticef(format string, args ...interface{}) {
	l.Warnf(format, args...)
}

// Notice bridges calls to the Go SDK logger's Warn.
func (l *Logger) Notice(args ...interface{}) {
	l.Warn(args...)
}

// Criticalf bridges calls to the Go SDK logger's Criticalf. .....
func (l *Logger) Criticalf(format string, args ...interface{}) {
	l.Warnf(format, args...)
}

// Critical bridges calls to the Go SDK logger's Critical.
func (l *Logger) Critical(args ...interface{}) {
	l.Warn(args...)
}

// IsEnabledFor bridges calls to the Go SDK logger's IsEnabledFor.
func (l *Logger) IsEnabledFor(level Level) bool {
	return logging.IsEnabledFor(logging.Level(level), l.Module)
}

// SetFormat
func SetFormat(formatSpec string) string {
	//do nothing
	return ""
}

// InitBackend
func InitBackend(formatter string, output io.Writer) {
	//do nothing
}

//InitFromSpec
func InitFromSpec(spec string) string {
	//do nothing
	return ""
}
