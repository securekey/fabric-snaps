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

	"github.com/stretchr/testify/require"

	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	mockstub "github.com/securekey/fabric-snaps/mocks/mockstub"
	"github.com/securekey/fabric-snaps/util/errors"
	"github.com/spf13/viper"

	"encoding/pem"

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

func TestAsync(t *testing.T) {
	httpService, err := Get(channelID)
	require.NoError(t, err)

	req := HTTPServiceInvokeRequest{
		RequestURL:     "https://localhost:8443/hello",
		RequestHeaders: headers,
		RequestBody:    jsonStr,
		NamedClient:    "abc",
		TxID:           "txid1",
	}

	t.Run("Valid request", func(t *testing.T) {
		invoker, err := httpService.NewInvoker(req)
		require.NoError(t, err)

		respChan, errChan := invoker.InvokeAsync()
		require.NoError(t, err)

		select {
		case resp := <-respChan:
			require.True(t, strings.Contains(string(resp), "Hello"))
		case err := <-errChan:
			t.Fatalf("Got error: %s", err)
		}
	})

	t.Run("Invalid URL", func(t *testing.T) {
		invalidReq := req
		invalidReq.RequestURL = ""

		_, err := httpService.NewInvoker(invalidReq)
		require.Error(t, err)
	})

	t.Run("Invalid named client", func(t *testing.T) {
		invalidReq := req
		invalidReq.NamedClient = "xyz"

		invoker, err := httpService.NewInvoker(invalidReq)
		require.NoError(t, err)

		respChan, errChan := invoker.InvokeAsync()
		require.NoError(t, err)

		select {
		case <-respChan:
			t.Fatalf("Expecting error")
		case err := <-errChan:
			t.Logf("Got expected error: %s", err)
		}
	})

	t.Run("Invalid response", func(t *testing.T) {
		invalidReq := req
		invalidReq.RequestURL = "https://localhost:8443/test/invalidJSONResponse"

		invoker, err := httpService.NewInvoker(invalidReq)
		require.NoError(t, err)

		respChan, errChan := invoker.InvokeAsync()
		require.NoError(t, err)

		select {
		case <-respChan:
			t.Fatalf("Expecting error")
		case err := <-errChan:
			t.Logf("Got expected error: %s", err)
		}
	})
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
		RequestBody: jsonStr}, "failed to validate response body against schema: description is required")

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
	var err error
	instance.config = &customHTTPConfig{Config: currentConfig, customCaCerts: []string{"cert1,cert2"}}
	instance.certPool, err = commtls.NewCertPool(currentConfig.IsSystemCertPoolEnabled())
	assert.Nil(t, err)
	verifyFailure(t, HTTPServiceInvokeRequest{RequestURL: "https://localhost:8443/hello", RequestHeaders: headers,
		RequestBody: jsonStr}, "certificate signed by unknown authority")

	// Failed path: invalid client key or cert
	instance.config = &customHTTPConfig{Config: currentConfig, customClientCert: "invalid.crt"}
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
	assert.NotEmpty(t, xCertPool.Subjects(), "cert pool not supposed to be empty, since `system cert pool = true` in config")
	baseNoOfSubjects := len(xCertPool.Subjects())

	//TEST CERT POOL OBJECT,
	// add a cert1 to certpool instance
	caCert1Bytes, err := ioutil.ReadFile(viper.GetString("http.tls.caCert.file"))
	if err != nil {
		t.Fatal("failed to get ca cert")
	}
	xCertPool.AppendCertsFromPEM(caCert1Bytes)

	//add cert2 to certpool instance
	caCert2Bytes, err := ioutil.ReadFile(viper.GetString("http.tls.cert.file"))
	if err != nil {
		t.Fatal("failed to get cert")
	}
	xCertPool.AppendCertsFromPEM(caCert2Bytes)

	xCertPool2, err := instance.certPool.Get()
	assert.Equal(t, len(xCertPool.Subjects()), len(xCertPool2.Subjects()), "any tampering to certpool returned will effect `instance.certPool`")

	//Add 2 new certs to httpsnap cert pool
	cert1, err := getCertFromPEMBytes(caCert1Bytes)
	assert.Nil(t, err)
	assert.NotNil(t, cert1)

	cert2, err := getCertFromPEMBytes(caCert2Bytes)
	assert.Nil(t, err)
	assert.NotNil(t, cert2)

	//now pool has baseNoOfSubjects + 2 certs
	instance.certPool.Add(cert1, cert2)
	xCertPool, err = instance.certPool.Get()
	assert.Equal(t, baseNoOfSubjects+2, len(xCertPool.Subjects()), "cert pool supposed to have certs")

	//Call init again and again
	initialize(config)

	//certs in cert pool will get reset, `cert1, cert2` being added previously should be gone
	xCertPool, err = instance.certPool.Get()
	assert.Equal(t, baseNoOfSubjects, len(xCertPool.Subjects()), "cert pool supposed to have certs")
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
		t.Fatalf("Expecting response to contain '%s', got '%s'", expected, string(res))
	}
}

func verifyFailure(t *testing.T, httpServiceInvokeRequest HTTPServiceInvokeRequest, expected string) {
	httpService, err := Get(channelID)
	if err != nil {
		t.Fatalf("Get return error: %s", err)
	}
	_, err = httpService.Invoke(httpServiceInvokeRequest)
	if err == nil {
		t.Fatalf("Invoke should have failed with err '%s'", expected)
	}
	if !strings.Contains(string(err.Error()), expected) {
		t.Fatalf("Expecting error response to contain '%s', got '%s'", expected, string(err.Error()))
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

type customHTTPConfig struct {
	api.Config
	customCaCerts    []string
	customClientCert string
}

func (c *customHTTPConfig) GetClientCert() (string, errors.Error) {
	if len(c.customClientCert) > 0 {
		return c.customClientCert, nil
	}
	return c.Config.GetClientCert()
}

func (c *customHTTPConfig) GetCaCerts() ([]string, errors.Error) {
	if len(c.customCaCerts) > 0 {
		return c.customCaCerts, nil
	}
	return c.Config.GetCaCerts()
}

func getCertFromPEMBytes(pemCerts []byte) (*x509.Certificate, error) {
	for len(pemCerts) > 0 {
		var block *pem.Block
		block, pemCerts = pem.Decode(pemCerts)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}

		return cert, nil
	}

	return nil, fmt.Errorf("Empty cert bytes")
}
