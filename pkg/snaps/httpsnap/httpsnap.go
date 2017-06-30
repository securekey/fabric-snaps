/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package httpsnap

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

	logging "github.com/op/go-logging"
	"github.com/xeipuuv/gojsonschema"

	config "github.com/securekey/fabric-snaps/cmd/config"

	shim "github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

var logger = logging.MustGetLogger("httpsnap")

//CCSnapImpl implementation
type CCSnapImpl struct {
}

// NewSnap - create new instance of snap
func NewSnap() shim.Chaincode {
	return &CCSnapImpl{}
}

// Init snap
func (es *CCSnapImpl) Init(stub shim.ChaincodeStubInterface) pb.Response {
	// Nothing to do
	return shim.Success(nil)
}

// Invoke snap
func (es *CCSnapImpl) Invoke(stub shim.ChaincodeStubInterface) pb.Response {

	args := stub.GetArgs()

	if len(args) < 3 {
		return shim.Error("Missing URL parameter, content type and/or request body")
	}

	requestURL := string(args[0])
	if requestURL == "" {
		return shim.Error("Missing URL parameter")
	}

	contentType := string(args[1])
	if contentType == "" {
		return shim.Error("Missing content type")
	}

	requestBody := string(args[2])
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
	if len(args) >= 5 && args[4] != nil && len(args[4]) > 0 && strings.TrimSpace(string(args[4])) != "" {
		pins = strings.Split(string(args[4]), ",")
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

	// Validate request body against schema
	if err := validateRequest(contentType, requestBody); err != nil {
		errMsg := fmt.Sprintf("Failed to validate request body: %s", err)
		logger.Infof(errMsg)
		return shim.Error(errMsg)
	}

	// URL is ok, retrieve data using http client
	responseContentType, response, err := getData(requestURL, contentType, requestBody, client, pins)
	if err != nil {
		logger.Error(err)
		return shim.Error(err.Error())
	}

	logger.Debugf("Successfully retrieved data from URL: %s", requestURL)

	// Validate response body against schema
	if err := validateResponse(responseContentType, string(response)); err != nil {
		logger.Infof("Failed to validate response body: %s", err)
		return shim.Error(err.Error())
	}

	return shim.Success(response)

}

func getData(url string, requestContentType string, requestBody string, namedClient string, pins []string) (responseContentType string, responseBody []byte, err error) {

	tlsConfig, err := getTLSConfig(namedClient)
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
		logger.Warning(errMsg)
		return "", nil, fmt.Errorf(errMsg)
	}

	responseContentType = resp.Header.Get("Content-Type")

	if requestContentType != responseContentType {
		errMsg := fmt.Sprintf("Response content-type: %s doesn't match request content-type: %s", responseContentType, requestContentType)
		logger.Warning(errMsg)
		return "", nil, fmt.Errorf(errMsg)
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Warning("Read contents failed. url=%s, err=%s", url, err)
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

func getTLSConfig(client string) (*tls.Config, error) {

	// Default values
	clientCert := config.HTTPSnapCertPath()
	clientKey := config.HTTPSnapKeyPath()
	caCerts := config.HTTPSnapCaCerts()

	if client != "" {
		clientCert = fmt.Sprintf("%s/%s/%s.crt", config.HTTPSnapNamedClientOverridePath(), client, client)
		clientKey = fmt.Sprintf("%s/%s/%s.key", config.HTTPSnapNamedClientOverridePath(), client, client)
		caCerts = []string{fmt.Sprintf("%s/%s/%s-ca.crt", config.HTTPSnapNamedClientOverridePath(), client, client)}
	}

	// Load client cert
	cert, err := tls.LoadX509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, err
	}

	// Load CA certs
	caCertPool := x509.NewCertPool()
	for _, cert := range caCerts {
		caCert, err := ioutil.ReadFile(cert)
		if err != nil {
			return nil, err
		}
		caCertPool.AppendCertsFromPEM(caCert)
	}

	// Setup HTTPS client
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}, nil

}

func validateRequest(contentType string, request string) error {
	return validate(contentType, config.HTTPSnapRequestSchema(contentType), request)
}

func validateResponse(contentType string, response string) error {
	return validate(contentType, config.HTTPSnapResponseSchema(contentType), response)
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

	schemaLoader := gojsonschema.NewReferenceLoader("file://" + jsonSchema)
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
