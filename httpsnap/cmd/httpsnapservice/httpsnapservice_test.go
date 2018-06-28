/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package httpsnapservice

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
	mockstub "github.com/securekey/fabric-snaps/mocks/mockstub"
	"github.com/spf13/viper"

	commtls "github.com/hyperledger/fabric-sdk-go/pkg/core/config/comm/tls"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/securekey/fabric-snaps/httpsnap/api"
	"github.com/securekey/fabric-snaps/httpsnap/cmd/config"
	"github.com/securekey/fabric-snaps/httpsnap/cmd/sampleconfig"
	"github.com/stretchr/testify/assert"
)

var jsonStr = `{"id":"123", "name": "Test Name"}`
var headers = map[string]string{
	"Content-Type":  "application/json",
	"Authorization": "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==",
}

var channelID = "testChannel"
var mspID = "Org1MSP"

func TestRequiredArg(t *testing.T) {
	// Missing RequestURL
	verifyFailure(t, HTTPServiceInvokeRequest{RequestURL: "", RequestHeaders: headers,
		RequestBody: jsonStr}, "Missing RequestURL")

	var invalidHeaders = map[string]string{}

	// Missing required ContentType header tests
	verifyFailure(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/hello", RequestHeaders: invalidHeaders,
		RequestBody: jsonStr}, "Missing request headers")

	invalidHeaders["Test-Header"] = "Test"
	verifyFailure(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/hello", RequestHeaders: invalidHeaders,
		RequestBody: jsonStr}, "Missing required content-type header")

	invalidHeaders["Content-Type"] = ""
	verifyFailure(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/hello", RequestHeaders: invalidHeaders,
		RequestBody: jsonStr}, "content-type header is empty")

	// Missing RequestBody
	verifyFailure(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/hello", RequestHeaders: headers,
		RequestBody: ""}, "Missing RequestBody")

}

func TestNamedClient(t *testing.T) {

	// Failed path: Use invalid named client 'xyz' to override default TLS settings
	verifyFailure(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/hello", RequestHeaders: headers,
		RequestBody: jsonStr, NamedClient: "xyz"}, "client[xyz] crt not found")

	// Happy path: Should get "Hello" back - use named client 'abc' to override default TLS settings
	verifySuccess(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/hello", RequestHeaders: headers,
		RequestBody: jsonStr, NamedClient: "abc"}, "Hello")

	// Happy path: Should get "Hello" back - empty named client is using default TLS settings
	verifySuccess(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/hello", RequestHeaders: headers,
		RequestBody: jsonStr, NamedClient: ""}, "Hello")

}

func TestCertPinning(t *testing.T) {

	// Happy path: Should get "Hello" back - one pin provided
	verifySuccess(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/hello", RequestHeaders: headers,
		RequestBody: jsonStr, PinSet: []string{"JimkpX4DHgDC5gzsmyfTSDuYi+qCAaW36LXrSqvoTHY="}}, "Hello")

	// Happy path: Should get "Hello" back - pinset is provided (comma separated)
	verifySuccess(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/hello", RequestHeaders: headers,
		RequestBody: jsonStr, PinSet: []string{"JimkpX4DHgDC5gzsmyfTSDuYi+qCAaW36LXrSqvoTHY=", "pin2"}}, "Hello")

	// Happy path: Should get "Hello" back - nil pinset is provided (no cert pin validation)
	verifySuccess(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/hello", RequestHeaders: headers,
		RequestBody: jsonStr, PinSet: nil}, "Hello")

	// Failed path: Invalid pinset is provided
	verifyFailure(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/hello", RequestHeaders: headers,
		RequestBody: jsonStr, PinSet: []string{"pin1", "pin2", "pin3"}}, "Failed to validate peer cert pins")
}

func TestJsonValidation(t *testing.T) {

	// Happy path: Validation is correct for both request and response (got "Hello" back)
	verifySuccess(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/hello", RequestHeaders: headers,
		RequestBody: jsonStr}, "Hello")

	// Failed path: Request fails schema validation
	invalidJSONStr := `{"test": "test"}`
	verifyFailure(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/test/abc", RequestHeaders: headers,
		RequestBody: string(invalidJSONStr)}, "Failed to validate request body: id is required, name is required")

	// Failed path: Response fails schema validation
	verifyFailure(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/test/invalidJSONResponse", RequestHeaders: headers,
		RequestBody: jsonStr}, "validate return error: description is required")

	// Failed path: Request content type doesn't match response content type
	verifyFailure(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/test/textResponse", RequestHeaders: headers,
		RequestBody: jsonStr}, "Response content-type: text/plain; charset=utf-8 doesn't match request content-type: application/json")

	// Failed path: Wrong request content type (not JSON)
	verifyFailure(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/hello", RequestHeaders: map[string]string{"Content-Type": "text/html"},
		RequestBody: jsonStr}, "text/html not found")

}

func TestPost(t *testing.T) {

	// Happy path: Should get "Hello" back - use default TLS settings
	verifySuccess(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/hello", RequestHeaders: headers,
		RequestBody: jsonStr}, "Hello")

	// Failed Path: Connect to Google
	verifyFailure(t, HTTPServiceInvokeRequest{RequestURL: "https://www.google.ca", RequestHeaders: headers,
		RequestBody: jsonStr}, "Method Not Allowed, url=https://www.google.ca")

	// Failed Path: Http Status NOT OK
	verifyFailure(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/test/statusNotOK", RequestHeaders: headers,
		RequestBody: jsonStr}, "status: 500")

	// Failed path - should get 404 back since there's no handler for xyz
	verifyFailure(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/test/xyz", RequestHeaders: headers,
		RequestBody: jsonStr}, "status: 404")

	currentConfig := instance.config
	currenCertPool := instance.certPool

	// Failed path: invalid ca
	instance.config = &customHttpConfig{Config: currentConfig, customCaCerts: []string{"cert1,cert2"}}
	instance.certPool = commtls.NewCertPool(currentConfig.IsSystemCertPoolEnabled())
	verifyFailure(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/hello", RequestHeaders: headers,
		RequestBody: jsonStr}, "certificate signed by unknown authority")

	// Failed path: invalid client key or cert
	instance.config = &customHttpConfig{Config: currentConfig, customClientCert: "invalid.crt"}
	verifyFailure(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/hello", RequestHeaders: headers,
		RequestBody: jsonStr}, "could not decode pem bytes")

	instance.config = currentConfig
	instance.certPool = currenCertPool

}

func TestHttpServiceRefresh(t *testing.T) {

	instance = &HTTPServiceImpl{}
	assert.Nil(t, instance.certPool, "cert pool is not supposed to be preinitialized")
	assert.Nil(t, instance.config, "config is not supposed to be preinitialized")

	config, _, err := config.NewConfig("../sampleconfig", "testChannel")
	assert.Nil(t, err, "not supposed to get error while creating httpsnap config")

	//call initialize
	initialize(config)
	assert.NotNil(t, instance.certPool, "cert pool is supposed to be not nil")
	assert.NotNil(t, instance.config, "config is supposed to be not nil")

	xCertPool, err := instance.certPool.Get()
	assert.Empty(t, xCertPool.Subjects(), "cert pool supposed to be empty")

	//add a cert1
	caCert, err := ioutil.ReadFile(viper.GetString("http.tls.caCert.file"))
	if err != nil {
		t.Fatal("failed to get ca cert")
	}
	xCertPool.AppendCertsFromPEM(caCert)

	//add cert2
	caCert, err = ioutil.ReadFile(viper.GetString("http.tls.cert.file"))
	if err != nil {
		t.Fatal("failed to get cert")
	}
	xCertPool.AppendCertsFromPEM(caCert)

	//now pool has 2 certs
	xCertPool, err = instance.certPool.Get()
	assert.Equal(t, 2, len(xCertPool.Subjects()), "cert pool supposed to have 2 certs")

	//Call init again and again
	initialize(config)

	//cert pool should reset on refresh after config update
	xCertPool, err = instance.certPool.Get()
	assert.Empty(t, xCertPool.Subjects(), "cert pool supposed to be empty")
}

func verifySuccess(t *testing.T, httpServiceInvokeRequest HTTPServiceInvokeRequest, expected string) {
	httpService, err := Get(channelID)
	if err != nil {
		t.Fatalf("Get return error: %s", err)
	}
	res, err := httpService.Invoke(httpServiceInvokeRequest)
	if err != nil {
		t.Fatalf("Invoke should have completed successfully: %s", err)
	}

	if !strings.Contains(string(res), expected) {
		t.Fatalf("Expecting response to contain %s, got %s", expected, string(res))
	}
}

func verifyFailure(t *testing.T, httpServiceInvokeRequest HTTPServiceInvokeRequest, expected string) {
	httpService, err := Get(channelID)
	if err != nil {
		t.Fatalf("Get return error: %s", err)
	}
	_, err = httpService.Invoke(httpServiceInvokeRequest)
	if err == nil {
		t.Fatalf("Invoke should have failed with err %s", expected)
	}
	if !strings.Contains(string(err.Error()), expected) {
		t.Fatalf("Expecting error response to contain %s, got %s", expected, string(err.Error()))
	}
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
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
	viper.Set("http.tls.caCert.file", "../test-data/httpserver/test-client.crt")
	viper.Set("http.tls.cert.file", "../test-data/httpserver/server.crt")
	viper.Set("http.tls.key.file", "../test-data/httpserver/server.key")

}

func TestMain(m *testing.M) {
	configData, err := ioutil.ReadFile("../sampleconfig/config.yaml")
	if err != nil {
		panic(fmt.Sprintf("File error: %s\n", err))
	}
	config := &configmanagerApi.ConfigMessage{MspID: mspID, Peers: []configmanagerApi.PeerConfig{configmanagerApi.PeerConfig{PeerID: "jdoe",
		App: []configmanagerApi.AppConfig{configmanagerApi.AppConfig{AppName: "httpsnap", Version: configmanagerApi.VERSION, Config: string(configData)}}}}}
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

	//Change peer config path
	PeerConfigPath = sampleconfig.ResolvPeerConfig("../sampleconfig")

	//Setup bccsp factory
	opts := sampleconfig.GetSampleBCCSPFactoryOpts("../sampleconfig")

	factory.InitFactories(opts)

	go startHTTPServer()

	// Allow HTTP server to start
	time.Sleep(2 * time.Second)

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

func newConfigMockStub(channelID string) *mockstub.MockStub {
	stub := mockstub.NewMockStub("testConfigState", nil)
	stub.SetMspID("Org1MSP")
	stub.MockTransactionStart("startTxn")
	stub.ChannelID = channelID
	return stub
}

type customHttpConfig struct {
	api.Config
	customCaCerts    []string
	customClientCert string
}

func (c *customHttpConfig) GetClientCert() (string, error) {
	if len(c.customClientCert) > 0 {
		return c.customClientCert, nil
	}
	return c.Config.GetClientCert()
}

func (c *customHttpConfig) GetCaCerts() ([]string, error) {
	if len(c.customCaCerts) > 0 {
		fmt.Println("XXXXXX c.customCaCerts", c.customCaCerts)
		return c.customCaCerts, nil
	}
	return c.Config.GetCaCerts()
}
