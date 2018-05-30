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
	httpsnapservice "github.com/securekey/fabric-snaps/httpsnap/cmd/httpsnapservice"
	mockstub "github.com/securekey/fabric-snaps/mocks/mockstub"

	"github.com/spf13/viper"

	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/securekey/fabric-snaps/httpsnap/cmd/sampleconfig"
)

var jsonStr = `{"id":"123", "name": "Test Name"}`
var contentType = "application/json"
var channelID = "testChannel"
var peerTLSChannelID = "testPeerTLSChannel"
var mspID = "Org1MSP"
var headers = map[string]string{
	"content-type":  "application/json",
	"authorization": "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==",
}

func TestInit(t *testing.T) {

	stub := newMockStub(channelID, mspID)

	res := stub.MockInit("txID", [][]byte{})
	if res.Status != shim.OK {
		t.Fatalf("Init failed: %v", res.Message)
	}
}

func TestInvalidParameters(t *testing.T) {

	stub := newMockStub(channelID, mspID)

	// Test required argument: function name
	testRequiredArg(t, stub, [][]byte{}, "function name")

	// Test required argument: request
	testRequiredArg(t, stub, [][]byte{[]byte("invoke")}, "Http Snap Request")

	// Required args: nil headers
	args := [][]byte{[]byte("invoke"), createHTTPSnapRequest("http://localhost/abc", nil, jsonStr)}
	verifyFailure(t, stub, args, "Invoke should have failed due to nil headers")

	// Required args: headers missing required 'Content-Type' header
	args = [][]byte{[]byte("invoke"), createHTTPSnapRequest("http://localhost/abc", map[string]string{}, jsonStr)}
	verifyFailure(t, stub, args, "Invoke should have failed due to missing required header")

	// Required args: empty Content-Type header
	args = [][]byte{[]byte("invoke"), createHTTPSnapRequest("http://localhost/abc", map[string]string{"content-type": ""}, jsonStr)}
	verifyFailure(t, stub, args, "Invoke should have failed due to empty content type")

	// Required args: empty request body
	args = [][]byte{[]byte("invoke"), createHTTPSnapRequest("http://localhost/abc", headers, "")}
	verifyFailure(t, stub, args, "Invoke should have failed due to empty request body")

	// Required args: empty URL
	args = [][]byte{[]byte("invoke"), createHTTPSnapRequest("", headers, jsonStr)}
	verifyFailure(t, stub, args, "Invoke should have failed due to empty URL")

	// Failed path: url syntax is not valid
	args = [][]byte{[]byte("invoke"), createHTTPSnapRequest("http/localhost/abc", headers, jsonStr)}
	verifyFailure(t, stub, args, "Invoke should have failed since URL syntax is not valid")

	// Failed path: HTTP url not allowed (only HTTPS)
	args = [][]byte{[]byte("invoke"), createHTTPSnapRequest("http://localhost/abc", headers, jsonStr)}
	verifyFailure(t, stub, args, "Invoke should have failed since URL doesn't start with https")
}

func TestUsingHttpService(t *testing.T) {

	stub := newMockStub(channelID, mspID)
	// Happy path: Should get "Hello" back - use default TLS settings
	args := [][]byte{[]byte("invoke"), createHTTPSnapRequest("https://localhost:8443/hello", headers, jsonStr)}
	verifySuccess(t, stub, args, "Hello")
	// Failed Path: Connect to Google
	args = [][]byte{[]byte("invoke"), createHTTPSnapRequest("https://www.google.ca", headers, jsonStr)}
	verifyFailure(t, stub, args, "Invoke should have failed to connect to google")

}

func TestUsingHttpServiceOnPeerTLSConfig(t *testing.T) {

	stub := newMockStub(peerTLSChannelID, mspID)
	// Happy path: Should get "Hello" back - use default TLS settings
	args := [][]byte{[]byte("invoke"), createHTTPSnapRequest("https://localhost:8443/hello", headers, jsonStr)}
	verifySuccess(t, stub, args, "Hello")
	// Failed Path: Connect to Google
	args = [][]byte{[]byte("invoke"), createHTTPSnapRequest("https://www.google.ca", headers, jsonStr)}
	verifyFailure(t, stub, args, "Invoke should have failed to connect to google")

}

func verifySuccess(t *testing.T, stub *mockstub.MockStub, args [][]byte, expected string) {
	res := stub.MockInvoke("txID", args)
	if res.Status != shim.OK {
		t.Fatalf("Invoke should have completed successfully args: %v", res.Message)
	}

	if !strings.Contains(string(res.Payload), expected) {
		t.Fatalf("Expecting response to contain %s, got %s", expected, string(res.Payload))
	}
}

func verifyFailure(t *testing.T, stub *mockstub.MockStub, args [][]byte, msg string) {
	res := stub.MockInvoke("txID", args)
	fmt.Println(res.Message)
	if res.Status == shim.OK {
		t.Fatalf("%s: %v", msg, res.Message)
	}

}

func testRequiredArg(t *testing.T, stub *mockstub.MockStub, args [][]byte, argName string) {

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
	config := &configmanagerApi.ConfigMessage{MspID: mspID, Peers: []configmanagerApi.PeerConfig{configmanagerApi.PeerConfig{PeerID: "jdoe",
		App: []configmanagerApi.AppConfig{configmanagerApi.AppConfig{AppName: "httpsnap", Version: configmanagerApi.VERSION, Config: string(configData)}}}}}
	stub := newConfigMockStub(channelID, mspID)
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

	//configdata for second channel for which peer TLS config is enabled
	configDataStr := string(configData)
	configDataStr = strings.Replace(configDataStr, "allowPeerConfig: false", "allowPeerConfig: true", -1)
	config2 := &configmanagerApi.ConfigMessage{MspID: mspID, Peers: []configmanagerApi.PeerConfig{configmanagerApi.PeerConfig{PeerID: "jdoe",
		App: []configmanagerApi.AppConfig{configmanagerApi.AppConfig{AppName: "httpsnap", Version: configmanagerApi.VERSION, Config: string(configDataStr)}}}}}
	configBytes2, err := json.Marshal(config2)
	if err != nil {
		panic(fmt.Sprintf("Cannot Marshal %s\n", err))
	}
	stub2 := newConfigMockStub(peerTLSChannelID, mspID)
	//upload valid message to HL
	err = uploadConfigToHL(stub2, configBytes2)
	if err != nil {
		panic(fmt.Sprintf("Cannot upload %s\n", err))
	}

	configmgmtService.Initialize(stub2, mspID)

	httpsnapservice.PeerConfigPath = sampleconfig.ResolvPeerConfig("./sampleconfig")

	go startHTTPServer()

	// Allow HTTP server to start
	time.Sleep(2 * time.Second)

	//Setup bccsp factory
	opts := sampleconfig.GetSampleBCCSPFactoryOpts("./sampleconfig")

	//Now call init factories using opts you got
	factory.InitFactories(opts)

	os.Exit(m.Run())
}

//uploadConfigToHL to upload key&config to repository
func uploadConfigToHL(stub *mockstub.MockStub, config []byte) error {
	configManager := mgmt.NewConfigManager(stub)
	if configManager == nil {
		return fmt.Errorf("Cannot instantiate config manager")
	}
	err := configManager.Save(config)
	return err

}

func createHTTPSnapRequest(url string, headers map[string]string, body string) []byte {

	req := api.HTTPSnapRequest{URL: url, Headers: headers, Body: body}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("err: %s\n", err)
		return nil
	}

	return reqBytes
}
