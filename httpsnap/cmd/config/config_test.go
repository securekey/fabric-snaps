/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"os"
	"strings"
	"testing"

	httpsnapApi "github.com/securekey/fabric-snaps/httpsnap/api"

	"github.com/spf13/viper"
)

var snapConfig *viper.Viper
var c httpsnapApi.Config

var relConfigPath = "/fabric-snaps/httpsnap/cmd/config/"

func TestGetClientCert(t *testing.T) {
	verifyEqual(t, c.GetClientCert(), snapConfig.GetString("tls.clientCert"), "Failed to get client cert.")
}
func TestGetClientKey(t *testing.T) {
	verifyEqual(t, c.GetClientKey(), snapConfig.GetString("tls.clientKey"), "Failed to get client key.")
}
func TestGetNamedClientOverridePath(t *testing.T) {
	verifyEqual(t, c.GetNamedClientOverridePath(), snapConfig.GetString("tls.namedClientOverridePath"), "Failed to get client override path.")
}

func TestGetShemaConfig(t *testing.T) {

	value, err := c.GetSchemaConfig("non-existent/type")
	if err == nil {
		t.Fatalf("Should have failed to retrieve schema config for non-existent type.")
	}

	expected := httpsnapApi.SchemaConfig{Type: "application/json", Request: "/schema/request.json", Response: "/schema/response.json"}
	value, err = c.GetSchemaConfig(expected.Type)

	if err != nil || *value != expected {
		t.Fatalf("Failed to get schema config. Expecting %s, got %s, err=%s ", expected, value, err)
	}
}

func TestGetCaCerts(t *testing.T) {
	values := c.GetCaCerts()
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
	var err error
	c, err = NewConfig("./", nil)
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
	configPath := c.GetConfigPath("/")
	if configPath != "/" {
		t.Fatalf(`Expected "/", got %s`, configPath)
	}

	// Test relative path
	configPath = c.GetConfigPath("rel/abc")
	expectedPath := relConfigPath + "rel/abc"
	if !strings.Contains(configPath, expectedPath) {
		t.Fatalf("Expecting response to contain %s, got %s", expectedPath, configPath)
	}
}

func TestNoConfig(t *testing.T) {
	viper.Reset()
	_, err := NewConfig("abc", nil)
	if err == nil {
		t.Fatalf("Init config should have failed.")
	}

}
