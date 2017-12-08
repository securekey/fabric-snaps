/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"

	"fmt"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/xeipuuv/gojsonschema"

	shim "github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/httpsnap/api"
	httpsnapConfig "github.com/securekey/fabric-snaps/httpsnap/cmd/config"
)

var logger = logging.NewLogger("httpsnap")

// peerConfigPath location of core.yaml
var peerConfigPath = ""

//HTTPSnap implementation
type HTTPSnap struct {
}

// New chaincode implementation
func New() shim.Chaincode {
	return &HTTPSnap{}
}

// Init snap
func (httpsnap *HTTPSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {

	logger.Info("Snap configuration loaded.")
	return shim.Success(nil)
}

// Invoke should be called with 4 mandatory arguments (and 2 optional ones):
// args[0] - Function (currently not used)
// args[1] - URL
// args[2] - Content-Type
// args[3] - Request Body
// args[4] - Named Client (optional)
// args[5] - Pin set (optional)
func (httpsnap *HTTPSnap) Invoke(stub shim.ChaincodeStubInterface) pb.Response {

	config, err := httpsnapConfig.NewConfig(peerConfigPath, stub.GetChannelID())
	if err != nil {
		errMsg := fmt.Sprintf("Failed to initialize config: %s", err)
		logger.Errorf(errMsg)
		return shim.Error(errMsg)
	}

	_, args := stub.GetFunctionAndParameters()

	if len(args) < 3 {
		return shim.Error("Missing URL parameter, content type and/or request body")
	}

	requestURL := args[0]
	if requestURL == "" {
		return shim.Error("Missing URL parameter")
	}

	contentType := args[1]
	if contentType == "" {
		return shim.Error("Missing content type")
	}

	requestBody := args[2]
	if requestBody == "" {
		return shim.Error("Missing request body")
	}

	// Optional parameter: named client (used for determining parameters for TLS configuration)
	client := ""
	if len(args) >= 4 {
		client = string(args[3])
	}

	// Optional parameter: pin set(comma separated)
	pins := []string{}
	if len(args) >= 5 && args[4] != "" && strings.TrimSpace(args[4]) != "" {
		pins = strings.Split(args[4], ",")
	}

	// Validate URL
	uri, err := url.ParseRequestURI(requestURL)
	if err != nil {
		errMsg := fmt.Sprintf("Invalid URL: %s", err.Error())
		logger.Infof(errMsg)
		return shim.Error(errMsg)
	}

	// Scheme has to be https
	if uri.Scheme != "https" {
		return shim.Error(fmt.Sprintf("Unsupported scheme: %s", uri.Scheme))
	}

	schemaConfig, err := config.GetSchemaConfig(contentType)
	if err != nil {
		logger.Error(err)
		return shim.Error(err.Error())
	}

	// Validate request body against schema
	if err := validate(contentType, schemaConfig.Request, requestBody); err != nil {
		errMsg := fmt.Sprintf("Failed to validate request body: %s", err)
		logger.Infof(errMsg)
		return shim.Error(errMsg)
	}

	// URL is ok, retrieve data using http client
	responseContentType, response, err := getData(requestURL, contentType, requestBody, client, pins, config)
	if err != nil {
		logger.Error(err)
		return shim.Error(err.Error())
	}

	logger.Debugf("Successfully retrieved data from URL: %s", requestURL)

	// Validate response body against schema
	if err := validate(responseContentType, schemaConfig.Response, string(response)); err != nil {
		logger.Infof("Failed to validate response body: %s", err)
		return shim.Error(err.Error())
	}

	return shim.Success(response)

}

func getData(url string, requestContentType string, requestBody string, namedClient string, pins []string, config api.Config) (responseContentType string, responseBody []byte, err error) {

	tlsConfig, err := getTLSConfig(namedClient, config)
	if err != nil {
		logger.Errorf("Failed to load tls config. namedClient=%s, err=%s", namedClient, err)
		return "", nil, err
	}

	tlsConfig.BuildNameToCertificate()
	var transport *http.Transport
	if len(pins) > 0 {
		transport = &http.Transport{TLSClientConfig: tlsConfig, DialTLS: verifyPinDialer(tlsConfig, pins)}
	} else {
		transport = &http.Transport{TLSClientConfig: tlsConfig}
	}

	client := &http.Client{Transport: transport}

	resp, err := client.Post(url, requestContentType, bytes.NewBuffer([]byte(requestBody)))
	if err != nil {
		logger.Errorf("POST failed. url=%s, err=%s", url, err)
		return "", nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Http response status code: %d, status: %s, url=%s", resp.StatusCode, resp.Status, url)
		logger.Warnln(errMsg)
		return "", nil, fmt.Errorf(errMsg)
	}

	responseContentType = resp.Header.Get("Content-Type")

	if requestContentType != responseContentType {
		errMsg := fmt.Sprintf("Response content-type: %s doesn't match request content-type: %s", responseContentType, requestContentType)
		logger.Warnln(errMsg)
		return "", nil, fmt.Errorf(errMsg)
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Warnf("Read contents failed. url=%s, err=%s", url, err)
		return "", nil, err
	}

	logger.Debugf("Got %s from url=%s", contents, url)

	return responseContentType, contents, nil
}

// Dialer is custom dialer to verify cert against pinset
type Dialer func(network, addr string) (net.Conn, error)

func verifyPinDialer(tlsConfig *tls.Config, pins []string) Dialer {

	return func(network, addr string) (net.Conn, error) {
		c, err := tls.Dial(network, addr, tlsConfig)
		if err != nil {
			return nil, err
		}

		var peerPins []string

		pinValid := false
		connState := c.ConnectionState()
		for _, peerCert := range connState.PeerCertificates {
			certPin := GeneratePin(peerCert)
			peerPins = append(peerPins, certPin)
			for _, pin := range pins {
				if pin == certPin {
					pinValid = true
					break
				}
			}
		}

		if pinValid == false {
			return nil, fmt.Errorf("Failed to validate peer cert pins %v against allowed pins: %v", peerPins, pins)
		}

		return c, nil
	}
}

// GeneratePin returns pin of an x509 certificate
func GeneratePin(c *x509.Certificate) string {
	digest := sha256.Sum256(c.RawSubjectPublicKeyInfo)
	return base64.StdEncoding.EncodeToString(digest[:])
}

func getTLSConfig(client string, config api.Config) (*tls.Config, error) {

	// Default values
	clientCert := config.GetClientCert()
	clientKey, err := config.GetClientKey()
	if err != nil {
		return nil, err
	}
	caCerts := config.GetCaCerts()

	if client != "" {
		clientOverrideCrtMap, err := config.GetNamedClientOverride()
		if err != nil {
			return nil, err
		}
		clientOverrideCrt := clientOverrideCrtMap[client]
		if clientOverrideCrt == nil {
			return nil, fmt.Errorf("client[%s] crt not found", client)
		}
		clientCert = clientOverrideCrt.Crt
		clientKey = clientOverrideCrt.Key
		caCerts = []string{clientOverrideCrt.Ca}
	}

	// Load client cert
	cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		return nil, err
	}

	// Load CA certs
	caCertPool := x509.NewCertPool()
	if config.IsSystemCertPoolEnabled() {
		var err error
		if caCertPool, err = x509.SystemCertPool(); err != nil {
			return nil, err
		}
		logger.Debugf("Loaded system cert pool of size: %d", len(caCertPool.Subjects()))
	}

	for _, cert := range caCerts {
		caCertPool.AppendCertsFromPEM([]byte(cert))
	}

	// Setup HTTPS client
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}, nil

}

func validate(contentType string, schema string, body string) error {

	switch contentType {
	case "application/json":
		return validateJSON(schema, body)
	default:
		return fmt.Errorf("Unsupported content type: '%s' ", contentType)
	}
}

func validateJSON(jsonSchema string, jsonStr string) error {
	logger.Debugf("Validating %s against schema: %s", jsonStr, jsonSchema)

	schemaLoader := gojsonschema.NewStringLoader(jsonSchema)
	result, err := gojsonschema.Validate(schemaLoader, gojsonschema.NewStringLoader(jsonStr))
	if err != nil {
		return err
	}

	if !result.Valid() {
		errMsg := ""
		for i, desc := range result.Errors() {
			errMsg += desc.Description()
			if i+1 < len(result.Errors()) {
				errMsg += ", "
			}
		}
		return errors.New(errMsg)

	}
	return nil
}

func main() {
}
