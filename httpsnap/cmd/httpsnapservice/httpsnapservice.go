/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package httpsnapservice

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"

	"github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/pkg/errors"
	httpsnapApi "github.com/securekey/fabric-snaps/httpsnap/api"
	httpsnapconfig "github.com/securekey/fabric-snaps/httpsnap/cmd/config"
	"github.com/xeipuuv/gojsonschema"
)

var logger = logging.NewLogger("http-service")

//PeerConfigPath use for testing
var PeerConfigPath = ""

//HTTPServiceImpl used to create transaction service
type HTTPServiceImpl struct {
	config httpsnapApi.Config
}

//HTTPServiceInvokeRequest used to create http invoke service
type HTTPServiceInvokeRequest struct {
	RequestURL  string
	ContentType string
	RequestBody string
	NamedClient string
	PinSet      []string
}

// Dialer is custom dialer to verify cert against pinset
type Dialer func(network, addr string) (net.Conn, error)

//Get will return httpService to caller
func Get(channelID string) (*HTTPServiceImpl, error) {
	return newHTTPService(channelID)
}

//Invoke http service
func (httpServiceImpl *HTTPServiceImpl) Invoke(httpServiceInvokeRequest HTTPServiceInvokeRequest) ([]byte, error) {
	if httpServiceInvokeRequest.RequestURL == "" {
		return nil, errors.New("Missing RequestURL")
	}
	if httpServiceInvokeRequest.ContentType == "" {
		return nil, errors.New("Missing ContentType")
	}
	if httpServiceInvokeRequest.RequestBody == "" {
		return nil, errors.New("Missing RequestBody")
	}

	// Validate URL
	uri, err := url.ParseRequestURI(httpServiceInvokeRequest.RequestURL)
	if err != nil {
		return nil, errors.Wrap(err, "Invalid URL")
	}

	// Scheme has to be https
	if uri.Scheme != "https" {
		return nil, errors.Errorf("Unsupported scheme: %s", uri.Scheme)
	}

	schemaConfig, err := httpServiceImpl.config.GetSchemaConfig(httpServiceInvokeRequest.ContentType)
	if err != nil {
		return nil, errors.WithMessage(err, "GetSchemaConfig return error")
	}

	// Validate request body against schema
	if err := httpServiceImpl.validate(httpServiceInvokeRequest.ContentType, schemaConfig.Request, httpServiceInvokeRequest.RequestBody); err != nil {
		return nil, errors.WithMessage(err, "Failed to validate request body")
	}

	// URL is ok, retrieve data using http client
	responseContentType, response, err := httpServiceImpl.getData(httpServiceInvokeRequest.RequestURL, httpServiceInvokeRequest.ContentType,
		httpServiceInvokeRequest.RequestBody, httpServiceInvokeRequest.NamedClient, httpServiceInvokeRequest.PinSet, httpServiceImpl.config)
	if err != nil {
		return nil, errors.WithMessage(err, "getData return error")
	}

	logger.Debugf("Successfully retrieved data from URL: %s", httpServiceInvokeRequest.RequestURL)

	// Validate response body against schema
	if err := httpServiceImpl.validate(responseContentType, schemaConfig.Response, string(response)); err != nil {
		return nil, errors.WithMessage(err, "validate return error")
	}
	return response, nil
}

//newHTTPService creates new http snap service
func newHTTPService(channelID string) (*HTTPServiceImpl, error) {
	config, err := httpsnapconfig.NewConfig(PeerConfigPath, channelID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize config")
	}

	if config == nil {
		return nil, errors.New("config from ledger is nil")
	}
	httpService := &HTTPServiceImpl{}
	httpService.config = config
	return httpService, nil

}

func (httpServiceImpl *HTTPServiceImpl) getData(url string, requestContentType string, requestBody string, namedClient string, pins []string, config httpsnapApi.Config) (responseContentType string, responseBody []byte, err error) {

	tlsConfig, err := httpServiceImpl.getTLSConfig(namedClient, config)
	if err != nil {
		logger.Errorf("Failed to load tls config. namedClient=%s, err=%s", namedClient, err)
		return "", nil, err
	}

	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSHandshakeTimeout: config.TimeoutOrDefault(httpsnapApi.TransportTLSHandshake),
		ResponseHeaderTimeout: config.TimeoutOrDefault(httpsnapApi.TransportResponseHeader),
		ExpectContinueTimeout: config.TimeoutOrDefault(httpsnapApi.TransportExpectContinue),
		IdleConnTimeout:       config.TimeoutOrDefault(httpsnapApi.TransportIdleConn),
		DisableCompression:    true,
		TLSClientConfig:       tlsConfig,
	}

	if len(pins) > 0 {
		transport.DialTLS = httpServiceImpl.verifyPinDialer(tlsConfig, pins, config)
	}

	client := &http.Client{
		Timeout:   config.TimeoutOrDefault(httpsnapApi.Global),
		Transport: transport,
	}

	logger.Debugf("Requesting %s from url=%s", requestBody, url)

	resp, err := client.Post(url, requestContentType, bytes.NewBuffer([]byte(requestBody)))
	if err != nil {
		logger.Errorf("POST failed. url=%s, err=%s", url, err)
		return "", nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, errors.Errorf("Http response status code: %d, status: %s, url=%s", resp.StatusCode, resp.Status, url)
	}

	responseContentType = resp.Header.Get("Content-Type")

	if requestContentType != responseContentType {
		return "", nil, errors.Errorf("Response content-type: %s doesn't match request content-type: %s", responseContentType, requestContentType)
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Warnf("Read contents failed. url=%s, err=%s", url, err)
		return "", nil, err
	}

	logger.Debugf("Got %s from url=%s", contents, url)

	return responseContentType, contents, nil
}

func (httpServiceImpl *HTTPServiceImpl) verifyPinDialer(tlsConfig *tls.Config, pins []string, config httpsnapApi.Config) Dialer {

	return func(network, addr string) (net.Conn, error) {

		d := &net.Dialer{
			Timeout:   config.TimeoutOrDefault(httpsnapApi.DialerTimeout),
			KeepAlive: config.TimeoutOrDefault(httpsnapApi.DialerKeepAlive),
		}

		c, err := tls.DialWithDialer(d, network, addr, tlsConfig)
		if err != nil {
			return nil, err
		}

		var peerPins []string

		pinValid := false
		connState := c.ConnectionState()
		for _, peerCert := range connState.PeerCertificates {
			certPin := httpServiceImpl.GeneratePin(peerCert)
			peerPins = append(peerPins, certPin)
			for _, pin := range pins {
				if pin == certPin {
					pinValid = true
					break
				}
			}
		}

		if pinValid == false {
			return nil, errors.Errorf("Failed to validate peer cert pins %v against allowed pins: %v", peerPins, pins)
		}

		return c, nil
	}
}

// GeneratePin returns pin of an x509 certificate
func (httpServiceImpl *HTTPServiceImpl) GeneratePin(c *x509.Certificate) string {
	digest := sha256.Sum256(c.RawSubjectPublicKeyInfo)
	return base64.StdEncoding.EncodeToString(digest[:])
}

func (httpServiceImpl *HTTPServiceImpl) getTLSConfig(client string, config httpsnapApi.Config) (*tls.Config, error) {

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
			return nil, errors.Errorf("client[%s] crt not found", client)
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

func (httpServiceImpl *HTTPServiceImpl) validate(contentType string, schema string, body string) error {

	switch contentType {
	case "application/json":
		return httpServiceImpl.validateJSON(schema, body)
	default:
		return errors.Errorf("Unsupported content type: '%s' ", contentType)
	}
}

func (httpServiceImpl *HTTPServiceImpl) validateJSON(jsonSchema string, jsonStr string) error {
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
