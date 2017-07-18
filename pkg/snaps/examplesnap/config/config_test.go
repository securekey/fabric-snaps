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

var expected = "Configured Hello"
var relConfigPath = "/fabric-snaps/pkg/snaps/examplesnap/sampleconfig/"

var snapConfig *viper.Viper

func TestGetGreeting(t *testing.T) {

	greeting := GetGreeting()
	expected := snapConfig.GetString("greeting")
	if greeting != expected {
		t.Fatalf("Get greeting failed. Expected %s, got %s", expected, greeting)
	}
}

func TestMain(m *testing.M) {
	err := Init("../sampleconfig")
	if err != nil {
		panic(err.Error())
	}

	snapConfig = viper.New()
	snapConfig.SetConfigFile("../sampleconfig/config.yaml")
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
