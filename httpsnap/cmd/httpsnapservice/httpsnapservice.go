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

	"encoding/pem"

	"crypto"
	"io"

	"crypto/rsa"

	"crypto/ecdsa"

	"github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
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

	bccspSuite := factory.GetDefault()

	var clientCert string
	var caCerts []string
	var pk bccsp.Key
	var err error

	if client != "" {
		//Use client TLS config override in https snap config
		clientOverrideCrtMap, err := config.GetNamedClientOverride()
		if err != nil {
			return nil, err
		}
		clientOverrideCrt := clientOverrideCrtMap[client]
		if clientOverrideCrt == nil {
			return nil, errors.Errorf("client[%s] crt not found", client)
		}

		clientCert = clientOverrideCrt.Crt
		caCerts = []string{clientOverrideCrt.Ca}

	} else {

		// Use default TLS config in https snap config
		clientCert, err = config.GetClientCert()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get client cert from httpsnap config")
		}

		caCerts, err = config.GetCaCerts()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get ca certs from httpsnap config")
		}

	}

	//Get Key from Pem bytes
	key, err := httpServiceImpl.getCryptoSuiteKeyFromPem([]byte(clientCert), bccspSuite)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get key from client cert")
	}

	//Get private key using SKI
	pk, err = bccspSuite.GetKey(key.SKI())

	if pk != nil {
		//If private key available then get tls config from private key
		return httpServiceImpl.prepareTLSConfigFromPrivateKey(bccspSuite, pk, clientCert, caCerts, config.IsSystemCertPoolEnabled())

	} else if config.IsPeerTLSConfigEnabled() {
		// If private key not found and userPeerConfig enabled, then use peer tls client key
		peerClientTLSKey, err := config.GetPeerClientKey()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get peer tls client key")
		}
		return httpServiceImpl.prepareTLSConfigFromClientKeyBytes(clientCert, peerClientTLSKey, caCerts, config.IsSystemCertPoolEnabled())

	}

	return nil, errors.Wrap(err, "failed to get private key from SKI")

}

func (httpServiceImpl *HTTPServiceImpl) prepareTLSConfigFromClientKeyBytes(clientCert, clientKey string, caCerts []string, systemCertPoolEnabled bool) (*tls.Config, error) {
	// Load client cert
	cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		return nil, err
	}

	// Load CA certs
	caCertPool := x509.NewCertPool()
	if systemCertPoolEnabled {
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

func (httpServiceImpl *HTTPServiceImpl) prepareTLSConfigFromPrivateKey(bccspSuite bccsp.BCCSP, clientKey bccsp.Key, clientCert string, caCerts []string, systemCertPoolEnabled bool) (*tls.Config, error) {
	// Load client cert
	tlscert, err := x509KeyPair([]byte(clientCert), clientKey, bccspSuite)
	if err != nil {
		return nil, err
	}

	// Load CA certs
	caCertPool := x509.NewCertPool()
	if systemCertPoolEnabled {
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
		Certificates: []tls.Certificate{tlscert},
		RootCAs:      caCertPool,
	}, nil
}

func x509KeyPair(certPEMBlock []byte, clientKey bccsp.Key, bccspSuite bccsp.BCCSP) (tls.Certificate, error) {

	fail := func(err error) (tls.Certificate, error) { return tls.Certificate{}, err }

	var cert tls.Certificate
	var skippedBlockTypes []string
	for {
		var certDERBlock *pem.Block
		certDERBlock, certPEMBlock = pem.Decode(certPEMBlock)
		if certDERBlock == nil {
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			cert.Certificate = append(cert.Certificate, certDERBlock.Bytes)
		} else {
			skippedBlockTypes = append(skippedBlockTypes, certDERBlock.Type)
		}
	}

	var err error
	// We are parsing public key for TLS to find its type
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fail(err)
	}

	switch x509Cert.PublicKey.(type) {
	case *rsa.PublicKey:
		cert.PrivateKey = &PrivateKey{bccspSuite, clientKey, &rsa.PublicKey{}}
	case *ecdsa.PublicKey:
		cert.PrivateKey = &PrivateKey{bccspSuite, clientKey, &ecdsa.PublicKey{}}
	default:
		return fail(errors.New("tls: unknown public key algorithm"))
	}

	return cert, nil
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

func (httpServiceImpl *HTTPServiceImpl) getCryptoSuiteKeyFromPem(idBytes []byte, cryptoSuite bccsp.BCCSP) (bccsp.Key, error) {
	if idBytes == nil {
		return nil, errors.New("getCryptoSuiteKeyFromPem error: nil idBytes")
	}

	// Decode the pem bytes
	pemCert, _ := pem.Decode(idBytes)
	if pemCert == nil {
		return nil, errors.Errorf("getCryptoSuiteKeyFromPem error: could not decode pem bytes [%v]", idBytes)
	}

	// get a cert
	cert, err := x509.ParseCertificate(pemCert.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "getCryptoSuiteKeyFromPem error: failed to parse x509 cert")
	}

	// get the public key in the right format
	certPubK, err := cryptoSuite.KeyImport(cert, &bccsp.X509PublicKeyImportOpts{Temporary: true})

	return certPubK, nil
}

//PrivateKey is signer implementation for golang client TLS
type PrivateKey struct {
	bccsp     bccsp.BCCSP
	key       bccsp.Key
	publicKey crypto.PublicKey
}

// Public returns the public key corresponding to priv.
func (priv *PrivateKey) Public() crypto.PublicKey {
	return priv.publicKey
}

// Sign signs msg with priv, reading randomness from rand. If opts is a
// *PSSOptions then the PSS algorithm will be used, otherwise PKCS#1 v1.5 will
// be used. This method is intended to support keys where the private part is
// kept in, for example, a hardware module. Common uses should use the Sign*
// functions in this package.
func (priv *PrivateKey) Sign(rand io.Reader, msg []byte, opts crypto.SignerOpts) ([]byte, error) {
	return priv.bccsp.Sign(priv.key, msg, opts)
}
