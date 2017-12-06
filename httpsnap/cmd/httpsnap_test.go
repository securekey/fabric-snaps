/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"

	"github.com/securekey/fabric-snaps/httpsnap/api"
	"github.com/spf13/viper"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var testHost = "0.0.0.0"
var testPort = 15484
var jsonStr = []byte(`{"id":"123", "name": "Test Name"}`)
var contentType = "application/json"
var config api.Config
var channelID = "testChannel"
var mspID = "Org1MSP"

func TestInit(t *testing.T) {

	stub := newMockStub(channelID)

	res := stub.MockInit("txID", [][]byte{})
	if res.Status != shim.OK {
		t.Fatalf("Init failed: %v", res.Message)
	}
}

func TestInvalidParameters(t *testing.T) {

	stub := newMockStub(channelID)

	// Test required argument: function name
	testRequiredArg(t, stub, [][]byte{}, "function name")

	// Test required argument: URL
	testRequiredArg(t, stub, [][]byte{}, "URL")

	// Test required argument: content type
	testRequiredArg(t, stub, [][]byte{[]byte("invoke"), []byte("http://localhost/abc")}, "content type")

	// Test required argument: request body
	testRequiredArg(t, stub, [][]byte{[]byte("invoke"), []byte("http://localhost/abc"), []byte(contentType)}, "request body")

	// Required args: empty URL
	args := [][]byte{[]byte("invoke"), []byte(""), []byte(contentType), []byte(jsonStr)}
	verifyFailure(t, stub, args, "Invoke should have failed due to empty URL")

	// Required args: empty content type
	args = [][]byte{[]byte("invoke"), []byte("http/localhost/abc"), []byte(""), []byte(jsonStr)}
	verifyFailure(t, stub, args, "Invoke should have failed due to empty content type")

	// Required args: empty request body
	args = [][]byte{[]byte("invoke"), []byte("http/localhost/abc"), []byte(contentType), []byte("")}
	verifyFailure(t, stub, args, "Invoke should have failed due to empty request body")

	// Failed path: url syntax is not valid
	args = [][]byte{[]byte("invoke"), []byte("http/localhost/abc"), []byte(contentType), []byte(jsonStr)}
	verifyFailure(t, stub, args, "Invoke should have failed since URL syntax is not valid")

	// Failed path: HTTP url not allowed (only HTTPS)
	args = [][]byte{[]byte("invoke"), []byte("http://localhost/abc"), []byte(contentType), []byte(jsonStr)}
	verifyFailure(t, stub, args, "Invoke should have failed since URL doesn't start with https")
}

func TestNamedClient(t *testing.T) {

	stub := newMockStub(channelID)

	// Failed path: Use invalid named client 'xyz' to override default TLS settings
	args := [][]byte{[]byte("invoke"), []byte("https://localhost:8443/hello"), []byte(contentType), []byte(jsonStr), []byte("xyz")}
	verifyFailure(t, stub, args, "Invoke should have failed due to invalid named client")

	// Happy path: Should get "Hello" back - use named client 'abc' to override default TLS settings
	args = [][]byte{[]byte("invoke"), []byte("https://localhost:8443/hello"), []byte(contentType), []byte(jsonStr), []byte("abc")}
	verifySuccess(t, stub, args, "Hello")

	// Happy path: Should get "Hello" back - nil named client is using default TLS settings
	args = [][]byte{[]byte("invoke"), []byte("https://localhost:8443/hello"), []byte(contentType), []byte(jsonStr), nil}
	verifySuccess(t, stub, args, "Hello")

	// Happy path: Should get "Hello" back - empty named client is using default TLS settings
	args = [][]byte{[]byte("invoke"), []byte("https://localhost:8443/hello"), []byte(contentType), []byte(jsonStr), []byte("")}
	verifySuccess(t, stub, args, "Hello")

}

func TestCertPinning(t *testing.T) {

	stub := newMockStub(channelID)

	// Happy path: Should get "Hello" back - one pin provided
	args := [][]byte{[]byte("invoke"), []byte("https://localhost:8443/hello"), []byte(contentType), []byte(jsonStr), []byte(""), []byte("c2MiEtoRw7m1kc2r4GnVCT89OxqXK24PFiK02Qo1PIs=")}
	verifySuccess(t, stub, args, "Hello")

	// Happy path: Should get "Hello" back - pinset is provided (comma separated)
	args = [][]byte{[]byte("invoke"), []byte("https://localhost:8443/hello"), []byte(contentType), []byte(jsonStr), []byte(""), []byte("c2MiEtoRw7m1kc2r4GnVCT89OxqXK24PFiK02Qo1PIs=,pin2")}
	verifySuccess(t, stub, args, "Hello")

	// Happy path: Should get "Hello" back - nil pinset is provided (no cert pin validation)
	args = [][]byte{[]byte("invoke"), []byte("https://localhost:8443/hello"), []byte(contentType), []byte(jsonStr), []byte(""), nil}
	verifySuccess(t, stub, args, "Hello")

	// Happy path: Should get "Hello" back - empty pinset is provided (no cert pin validation)
	args = [][]byte{[]byte("invoke"), []byte("https://localhost:8443/hello"), []byte(contentType), []byte(jsonStr), []byte(""), []byte("")}
	verifySuccess(t, stub, args, "Hello")

	// Failed path: Invalid pinset is provided
	args = [][]byte{[]byte("invoke"), []byte("https://localhost:8443/hello"), []byte(contentType), []byte(jsonStr), []byte(""), []byte("pin1,pin2,pin3")}
	verifyFailure(t, stub, args, "Invoke should have failed due to pin validation against provided pinset")
}

func TestJsonValidation(t *testing.T) {

	stub := newMockStub(channelID)

	// Happy path: Validation is correct for both request and response (got "Hello" back)
	args := [][]byte{[]byte("invoke"), []byte("https://localhost:8443/hello"), []byte(contentType), []byte(jsonStr)}
	verifySuccess(t, stub, args, "Hello")

	// Failed path: Request fails schema validation
	invalidJSONStr := `{"test": "test"}`
	args = [][]byte{[]byte("invoke"), []byte("https://localhost:8443/test/abc"), []byte(contentType), []byte(invalidJSONStr)}
	verifyFailure(t, stub, args, "Invoke should have failed to validate request schema")

	// Failed path: Response fails schema validation
	args = [][]byte{[]byte("invoke"), []byte("https://localhost:8443/test/invalidJSONResponse"), []byte(contentType), []byte(jsonStr)}
	verifyFailure(t, stub, args, "Invoke should have failed to validate response schema")

	// Failed path: Request content type doesn't match response content type
	args = [][]byte{[]byte("invoke"), []byte("https://localhost:8443/test/textResponse"), []byte(contentType), []byte(jsonStr)}
	verifyFailure(t, stub, args, "Invoke should have failed to validate response content type")

	// Failed path: Wrong request content type (not JSON)
	args = [][]byte{[]byte("invoke"), []byte("https://localhost:8443/hello"), []byte("text/html"), []byte(jsonStr)}
	verifyFailure(t, stub, args, "Invoke should have failed with unsupported content-type")

}

func TestPost(t *testing.T) {

	stub := newMockStub(channelID)

	// Happy path: Should get "Hello" back - use default TLS settings
	args := [][]byte{[]byte("invoke"), []byte("https://localhost:8443/hello"), []byte(contentType), []byte(jsonStr)}
	verifySuccess(t, stub, args, "Hello")

	// Failed Path: Connect to Google
	args = [][]byte{[]byte("invoke"), []byte("https://www.google.ca"), []byte(contentType), []byte(jsonStr)}
	verifyFailure(t, stub, args, "Invoke should have failed to connect to google")

	// Failed Path: Http Status NOT OK
	args = [][]byte{[]byte("invoke"), []byte("https://localhost:8443/test/statusNotOK"), []byte(contentType), []byte(jsonStr)}
	if res := stub.MockInvoke("txID", args); res.Status == shim.OK {
		t.Fatalf("Invoke should have failed with HTTP 500 : %v", res.Message)
	} else if !strings.Contains(res.Message, "500") {
		t.Fatalf("Expecting 500 message, got %s", res.Message)
	}

	// Failed path - should get 404 back since there's no handler for xyz
	args = [][]byte{[]byte("invoke"), []byte("https://localhost:8443/test/xyz"), []byte(contentType), []byte(jsonStr)}
	if res := stub.MockInvoke("txID", args); res.Status == shim.OK {
		t.Fatalf("Invoke should have failed since URL doesn't exist")
	} else if !strings.Contains(res.Message, "404") {
		t.Fatalf("Expecting 404 message, got %s", res.Message)
	}

	// Failed path: invalid ca
	os.Setenv("CORE_TLS_CACERTS", "cert1,cert2")
	args = [][]byte{[]byte("invoke"), []byte("https://localhost:8443/hello"), []byte(contentType), []byte(jsonStr)}
	verifyFailure(t, stub, args, "Invoke should have failed due to invalid ca cert pool")

	// Failed path: invalid client key or cert
	viper.Set("tls.clientCert", "invalid.crt")
	args = [][]byte{[]byte("invoke"), []byte("https://localhost:8443/hello"), []byte(contentType), []byte(jsonStr)}
	verifyFailure(t, stub, args, "Invoke should have failed due to invalid client cert")

}

func verifySuccess(t *testing.T, stub *shim.MockStub, args [][]byte, expected string) {
	res := stub.MockInvoke("txID", args)
	if res.Status != shim.OK {
		t.Fatalf("Invoke should have completed successfully args: %v", res.Message)
	}

	if !strings.Contains(string(res.Payload), expected) {
		t.Fatalf("Expecting response to contain %s, got %s", expected, string(res.Payload))
	}
}

func verifyFailure(t *testing.T, stub *shim.MockStub, args [][]byte, msg string) {
	res := stub.MockInvoke("txID", args)
	if res.Status == shim.OK {
		t.Fatalf("%s: %v", msg, res.Message)
	}
}

func testRequiredArg(t *testing.T, stub *shim.MockStub, args [][]byte, argName string) {

	// Test missing argument
	verifyFailure(t, stub, args, fmt.Sprintf("Should have failed due missing %s", argName))

	// Test nil argument
	verifyFailure(t, stub, append(args, nil), fmt.Sprintf("Should have failed due to nil %s", argName))

	// Test empty argument
	verifyFailure(t, stub, append(args, []byte("")), fmt.Sprintf("Should have failed due to empty %s", argName))
}

func startHTTPServer() {

	initHTTPServerConfig()

	// Register request handlers
	http.HandleFunc("/hello", HelloServer)
	http.HandleFunc("/test/invalidJSONResponse", InvalidJSONResponseServer)
	http.HandleFunc("/test/textResponse", TextServer)
	http.HandleFunc("/test/statusNotOK", StatusNotOKServer)

	caCert, err := ioutil.ReadFile(viper.GetString("http.tls.caCert.file"))
	if err != nil {
		fmt.Println("HTTP Server: Failed to read ca-cert. " + err.Error())
		return
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		ClientCAs:  caCertPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}
	tlsConfig.BuildNameToCertificate()

	server := &http.Server{
		Addr:      viper.GetString("http.listen.address"),
		TLSConfig: tlsConfig,
	}

	err = server.ListenAndServeTLS(viper.GetString("http.tls.cert.file"), viper.GetString("http.tls.key.file"))

	if err != nil {
		fmt.Println("HTTP Server: Failed to start. " + err.Error())
	}

}

// HelloServer greeting (JSON)
func HelloServer(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"description": "Hello"}`)
}

// InvalidJSONResponseServer greeting (invalid JSON response, should fail against response schema)
func InvalidJSONResponseServer(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"test": "test"}`)

}

// StatusNotOKServer greeting (return HTTP 500)
func StatusNotOKServer(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	io.WriteString(w, "Error")
}

// TextServer greeting (Content type is not JSON, it is text)
func TextServer(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "Hello")
}

func initHTTPServerConfig() {

	viper.Set("http.listen.address", "127.0.0.1:8443")
	viper.Set("http.tls.caCert.file", "./test-data/httpserver/test-client.crt")
	viper.Set("http.tls.cert.file", "./test-data/httpserver/server.crt")
	viper.Set("http.tls.key.file", "./test-data/httpserver/server.key")

}

func TestMain(m *testing.M) {
	configData, err := ioutil.ReadFile("./sampleconfig/config.yaml")
	if err != nil {
		panic(fmt.Sprintf("File error: %v\n", err))
	}
	config := &configmanagerApi.ConfigMessage{MspID: mspID, Peers: []configmanagerApi.PeerConfig{configmanagerApi.PeerConfig{PeerID: "jdoe", App: []configmanagerApi.AppConfig{configmanagerApi.AppConfig{AppName: "httpsnap", Config: string(configData)}}}}}
	stub := newConfigMockStub(channelID)
	configBytes, err := json.Marshal(config)
	if err != nil {
		panic(fmt.Sprintf("Cannot Marshal %s\n", err))
	}
	//upload valid message to HL
	err = uploadConfigToHL(stub, configBytes)
	if err != nil {
		panic(fmt.Sprintf("Cannot upload %s\n", err))
	}
	configmgmtService.Initialize(stub, mspID)
	peerConfigPath = "./sampleconfig"

	go startHTTPServer()

	// Allow HTTP server to start
	time.Sleep(2 * time.Second)

	os.Exit(m.Run())
}

//uploadConfigToHL to upload key&config to repository
func uploadConfigToHL(stub *shim.MockStub, config []byte) error {
	configManager := mgmt.NewConfigManager(stub)
	if configManager == nil {
		return fmt.Errorf("Cannot instantiate config manager")
	}
	err := configManager.Save(config)
	return err

}
