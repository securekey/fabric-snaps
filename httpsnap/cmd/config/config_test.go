/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

var snapConfig *viper.Viper

var relConfigPath = "/fabric-snaps/httpsnap/cmd/config/"

func TestGetClientCert(t *testing.T) {
	verifyEqual(t, GetClientCert(), snapConfig.GetString("tls.clientCert"), "Failed to get client cert.")
}
func TestGetClientKey(t *testing.T) {
	verifyEqual(t, GetClientKey(), snapConfig.GetString("tls.clientKey"), "Failed to get client key.")
}
func TestGetNamedClientOverridePath(t *testing.T) {
	verifyEqual(t, GetNamedClientOverridePath(), snapConfig.GetString("tls.namedClientOverridePath"), "Failed to get client override path.")
}

func TestGetShemaConfig(t *testing.T) {

	value, err := GetSchemaConfig("non-existent/type")
	if err == nil {
		t.Fatalf("Should have failed to retrieve schema config for non-existent type.")
	}

	expected := SchemaConfig{Type: "application/json", Request: "/schema/request.json", Response: "/schema/response.json"}
	value, err = GetSchemaConfig(expected.Type)

	if err != nil || *value != expected {
		t.Fatalf("Failed to get schema config. Expecting %s, got %s, err=%s ", expected, value, err)
	}
}

func TestGetCaCerts(t *testing.T) {
	values := GetCaCerts()
	if len(values) != 2 {
		t.Fatalf("Expecting 2 certs, got %d", len(values))
	}
}

func verifyEqual(t *testing.T, value string, expected string, errMsg string) {
	if value != expected {
		t.Fatalf("%s. Expecting %s, got %s", errMsg, expected, value)
	}
}

func TestMain(m *testing.M) {
	err := Init("./")
	if err != nil {
		panic(err.Error())
	}

	snapConfig = viper.New()
	snapConfig.SetConfigFile("./config.yaml")
	snapConfig.ReadInConfig()

	os.Exit(m.Run())
}

func TestGetConfigPath(t *testing.T) {

	// Test absolute path
	configPath := GetConfigPath("/")
	if configPath != "/" {
		t.Fatalf(`Expected "/", got %s`, configPath)
	}

	// Test relative path
	configPath = GetConfigPath("rel/abc")
	expectedPath := relConfigPath + "rel/abc"
	if !strings.Contains(configPath, expectedPath) {
		t.Fatalf("Expecting response to contain %s, got %s", expectedPath, configPath)
	}
}

func TestInitializeLogging(t *testing.T) {
	viper.Set("logging.level", "wrongLogValue")
	defer viper.Set("logging.level", "info")
	err := initializeLogging()
	if err == nil {
		t.Fatal("initializeLogging() didn't return error")
	}
	if err.Error() != "Error initializing log level: logger: invalid log level" {
		t.Fatal("initializeLogging() didn't return expected error msg")
	}
}

func TestNoConfig(t *testing.T) {
	viper.Reset()
	err := Init("abc")
	if err == nil {
		t.Fatalf("Init config should have failed.")
	}

}
