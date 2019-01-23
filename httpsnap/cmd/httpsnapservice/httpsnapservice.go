/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package httpsnapservice

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/common/verifier"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	commtls "github.com/hyperledger/fabric-sdk-go/pkg/core/config/comm/tls"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	httpsnapApi "github.com/securekey/fabric-snaps/httpsnap/api"
	httpsnapconfig "github.com/securekey/fabric-snaps/httpsnap/cmd/config"
	"github.com/securekey/fabric-snaps/util/errors"
	"github.com/xeipuuv/gojsonschema"
)

var logger = logging.NewLogger("httpsnap")

//PeerConfigPath use for testing
var PeerConfigPath = ""

var once sync.Once
var instance *HTTPServiceImpl

//HTTPServiceImpl used to create transaction service
type HTTPServiceImpl struct {
	sync.RWMutex
	config   httpsnapApi.Config
	certPool fab.CertPool
	metrics  *Metrics
}

//HTTPServiceInvokeRequest used to create http invoke service
type HTTPServiceInvokeRequest struct {
	RequestURL     string
	RequestHeaders map[string]string
	RequestBody    string
	NamedClient    string
	PinSet         []string

	// TxID is used only for logging
	TxID string
}

// Invoker invokes the HTTP service either synchronously or asynchronously.
type Invoker interface {
	// Invoke invokes the HTTP service synchronously and returns the response or error
	Invoke() ([]byte, errors.Error)

	// InvokeAsync invokes the HTTP service asynchronously and returns a response channel and error channel,
	// one of which will return the result of the invocation
	InvokeAsync() (chan []byte, chan errors.Error)
}

const (
	contentType = "content-type"
)

// Dialer is custom dialer to verify cert against pinset
type Dialer func(network, addr string) (net.Conn, error)

//Get will return httpService to caller
func Get(channelID string, metrics *Metrics) (*HTTPServiceImpl, error) {
	return newHTTPService(channelID, metrics)
}

//updateConfig http service updates http service config if provided config has any updates
func initialize(config httpsnapApi.Config) error {

	//Update config in httpServiceImpl if any config update found in new config
	instance.Lock()
	defer instance.Unlock()

	var err error
	instance.config = config
	instance.certPool, err = commtls.NewCertPool(config.IsSystemCertPoolEnabled())
	return err
}

//Invoke http service
func (httpServiceImpl *HTTPServiceImpl) Invoke(httpServiceInvokeRequest HTTPServiceInvokeRequest) ([]byte, errors.Error) {
	invoker, err := httpServiceImpl.NewInvoker(httpServiceInvokeRequest)
	if err != nil {
		return nil, err
	}
	return invoker.Invoke()
}

// NewInvoker returns a new Invoker
func (httpServiceImpl *HTTPServiceImpl) NewInvoker(httpServiceInvokeRequest HTTPServiceInvokeRequest) (Invoker, errors.Error) {
	if httpServiceInvokeRequest.RequestURL == "" {
		return nil, errors.New(errors.MissingRequiredParameterError, "Missing RequestURL")
	}

	if len(httpServiceInvokeRequest.RequestHeaders) == 0 {
		return nil, errors.New(errors.MissingRequiredParameterError, "Missing request headers")
	}
	headers, headerErr := httpServiceImpl.validateHeaders(httpServiceInvokeRequest)
	if headerErr != nil {
		return nil, errors.Errorf(errors.MissingRequiredParameterError, "Required parameter is missing: %s", headerErr)
	}

	httpServiceInvokeRequest.RequestHeaders = headers

	if httpServiceInvokeRequest.RequestBody == "" {
		return nil, errors.New(errors.MissingRequiredParameterError, "Missing RequestBody")
	}

	// Validate URL
	err := httpServiceImpl.validateURL(httpServiceInvokeRequest)
	if err != nil {
		return nil, err
	}
	httpServiceImpl.RLock()
	defer httpServiceImpl.RUnlock()

	schemaConfig, err := httpServiceImpl.config.GetSchemaConfig(headers[contentType])
	if err != nil {
		return nil, errors.WithMessage(errors.ValidationError, err, "GetSchemaConfig return error")
	}

	// Validate request body against schema
	if codedErr := httpServiceImpl.validate(headers[contentType], schemaConfig.Request, httpServiceInvokeRequest.RequestBody); codedErr != nil {
		return nil, errors.WithMessage(errors.ValidationError, codedErr, "Failed to validate request body")
	}

	return &invoker{
		service:      httpServiceImpl,
		request:      httpServiceInvokeRequest,
		schemaConfig: schemaConfig,
		headers:      headers,
	}, nil
}
func (httpServiceImpl *HTTPServiceImpl) validateHeaders(httpServiceInvokeRequest HTTPServiceInvokeRequest) (map[string]string, errors.Error) {
	headers := make(map[string]string)
	// Converting header names to lowercase
	for name, value := range httpServiceInvokeRequest.RequestHeaders {
		headers[strings.ToLower(name)] = value
	}

	if _, ok := headers[contentType]; !ok {
		return nil, errors.New(errors.MissingRequiredParameterError, "Missing required content-type header")
	}

	if val, ok := headers[contentType]; ok && val == "" {
		return nil, errors.New(errors.MissingRequiredParameterError, "content-type header is empty")
	}
	return headers, nil
}
func (httpServiceImpl *HTTPServiceImpl) validateURL(httpServiceInvokeRequest HTTPServiceInvokeRequest) errors.Error {
	// Validate URL
	uri, err := url.ParseRequestURI(httpServiceInvokeRequest.RequestURL)
	if err != nil {
		return errors.Wrap(errors.ValidationError, err, "Invalid URL")
	}

	// Security controls should be added by the chaincode that calls the HTTP snap

	// Scheme has to be https
	if uri.Scheme != "https" {
		return errors.Errorf(errors.ValidationError, "Unsupported scheme: %s", uri.Scheme)
	}
	return nil
}

//newHTTPService creates new http snap service
func newHTTPService(channelID string, metrics *Metrics) (*HTTPServiceImpl, error) {
	config, dirty, err := httpsnapconfig.NewConfig(PeerConfigPath, channelID)
	if err != nil {
		return nil, errors.Wrap(errors.InitializeConfigError, err, "Failed to initialize config")
	}

	if config == nil {
		return nil, errors.New(errors.InitializeConfigError, "config from ledger is nil")
	}

	once.Do(func() {
		instance = &HTTPServiceImpl{metrics: metrics}
		err = initialize(config)
		if err != nil {
			return
		}
		dirty = false
		logger.Infof("Created HTTPServiceImpl instance %v", time.Unix(time.Now().Unix(), 0))
	})

	if err != nil {
		return nil, errors.Wrap(errors.InitializeConfigError, err, "failed to initialize httpservice with new config")
	}

	if dirty {
		err = initialize(config)
		if err != nil {
			return nil, errors.Wrap(errors.InitializeConfigError, err, "failed to update httpservice with new config")
		}
	}
	return instance, nil
}

func (httpServiceImpl *HTTPServiceImpl) getData(invokeReq HTTPServiceInvokeRequest) (responseContentType string, responseBody []byte, codedErr errors.Error) {
	startTime := time.Now()
	defer func() { httpServiceImpl.metrics.HTTPTimer.Observe(time.Since(startTime).Seconds()) }()
	return httpServiceImpl.getDataFromSource(invokeReq)
}

func (httpServiceImpl *HTTPServiceImpl) getHTTPClient(invokeReq HTTPServiceInvokeRequest, forceReload bool) (*http.Client, error) {

	tlsConfig, codedErr := httpServiceImpl.getTLSConfig(invokeReq.NamedClient, httpServiceImpl.config, forceReload)
	if codedErr != nil {
		logger.Errorf("Failed to load tls config. namedClient=%s, err=%s", invokeReq.NamedClient, codedErr.GenerateLogMsg())
		return nil, codedErr
	}

	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSHandshakeTimeout: httpServiceImpl.config.TimeoutOrDefault(httpsnapApi.TransportTLSHandshake),
		ResponseHeaderTimeout: httpServiceImpl.config.TimeoutOrDefault(httpsnapApi.TransportResponseHeader),
		ExpectContinueTimeout: httpServiceImpl.config.TimeoutOrDefault(httpsnapApi.TransportExpectContinue),
		IdleConnTimeout:       httpServiceImpl.config.TimeoutOrDefault(httpsnapApi.TransportIdleConn),
		DisableCompression:    true,
		TLSClientConfig:       tlsConfig,
	}

	if len(invokeReq.PinSet) > 0 {
		transport.DialTLS = httpServiceImpl.verifyPinDialer(tlsConfig, invokeReq.PinSet, httpServiceImpl.config)
	}

	client := &http.Client{
		Timeout:   httpServiceImpl.config.TimeoutOrDefault(httpsnapApi.Global),
		Transport: transport,
	}

	return client, nil
}

func (httpServiceImpl *HTTPServiceImpl) getDataFromSource(invokeReq HTTPServiceInvokeRequest) (responseContentType string, responseBody []byte, codedErr errors.Error) {
	httpServiceImpl.metrics.HTTPCounter.Add(1)

	logger.Debugf("Requesting %s from url=%s", invokeReq.RequestBody, invokeReq.RequestURL)

	req, err := httpServiceImpl.prepareHTTPRequest(invokeReq)
	if err != nil {
		return "", nil, errors.Wrap(errors.ValidationError, err, "Failed http.NewRequest")
	}

	sendHTTPRequest := func(reload bool) (*http.Response, error) {
		client, err := httpServiceImpl.getHTTPClient(invokeReq, reload)
		if err != nil {
			logger.Errorf("Failed to load tls config. namedClient=%s, err=%s", invokeReq.NamedClient, err.Error())
			return nil, err
		}
		return client.Do(req)
	}

	resp, err := sendHTTPRequest(false)
	if err != nil && httpServiceImpl.config.IsKeyCacheEnabled() {
		//try again once by forcing cache reload
		logger.Debugf("Failed to send http request to URL '%s' due to error [%s], retrying with forced cache reload", invokeReq.RequestURL, err.Error())
		resp, err = sendHTTPRequest(true)
	}

	//if fails again then return error
	if err != nil {
		errObj := errors.Wrapf(errors.HTTPClientError, err, "POST failed. url=%s", invokeReq.RequestURL)
		logger.Errorf(errObj.GenerateLogMsg())
		return "", nil, errObj
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, errors.Errorf(errors.HTTPClientError, "Http response status code: %d, status: %s, url=%s", resp.StatusCode, resp.Status, invokeReq.RequestURL)
	}

	responseContentType = resp.Header.Get(contentType)

	if !strings.Contains(strings.ToLower(responseContentType), strings.ToLower(invokeReq.RequestHeaders[contentType])) {
		return "", nil, errors.Errorf(errors.ValidationError, "Response content-type: %s doesn't match request content-type: %s", responseContentType, invokeReq.RequestHeaders[contentType])
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Warnf("Read contents failed. url=%s, err=%s", invokeReq.RequestURL, err)
		return "", nil, errors.Wrapf(errors.SystemError, err, "Read contents failed. url=%s", invokeReq.RequestURL)
	}

	logger.Debugf("Got %s from url=%s", contents, invokeReq.RequestURL)

	return responseContentType, contents, nil
}

func (httpServiceImpl *HTTPServiceImpl) prepareHTTPRequest(invokeReq HTTPServiceInvokeRequest) (*http.Request, error) {

	req, err := http.NewRequest("POST", invokeReq.RequestURL, bytes.NewBuffer([]byte(invokeReq.RequestBody)))
	if err != nil {
		return nil, err
	}

	// Set allowed headers only
	for name, value := range invokeReq.RequestHeaders {
		allowed := httpServiceImpl.config.IsHeaderAllowed(name)
		if allowed {
			req.Header.Set(name, value)
			logger.Debugf("Setting header '%s' to '%s'", name, value)
		}
	}

	return req, nil
}

func (httpServiceImpl *HTTPServiceImpl) verifyPinDialer(tlsConfig *tls.Config, pins []string, config httpsnapApi.Config) Dialer {

	timeout := config.TimeoutOrDefault(httpsnapApi.DialerTimeout)
	keepAlive := config.TimeoutOrDefault(httpsnapApi.DialerKeepAlive)

	return func(network, addr string) (net.Conn, error) {

		d := &net.Dialer{
			Timeout:   timeout,
			KeepAlive: keepAlive,
		}

		c, err := tls.DialWithDialer(d, network, addr, tlsConfig)
		if err != nil {
			return nil, errors.Wrap(errors.HTTPClientError, err, "Failed tls.DialWithDialer")
		}

		chain := c.ConnectionState().VerifiedChains[0]
		if len(chain) == 0 {
			return nil, errors.Errorf(errors.InvalidCertPinError, "Verified certificate chain is empty")
		}

		leafCert := chain[0]

		// use the first in the list as client cert
		// compute the fingerprint of the cert
		fingerprint := httpServiceImpl.GeneratePin(leafCert)

		for _, pin := range pins {
			if pin == fingerprint {
				return c, nil
			}
		}

		return nil, errors.Errorf(errors.InvalidCertPinError, "Failed to validate peer cert %s against allowed pins: %s", fingerprint, pins)
	}
}

// GeneratePin returns pin of an x509 certificate
func (httpServiceImpl *HTTPServiceImpl) GeneratePin(c *x509.Certificate) string {
	digest := sha256.Sum256(c.RawSubjectPublicKeyInfo)
	return base64.StdEncoding.EncodeToString(digest[:])
}

func (httpServiceImpl *HTTPServiceImpl) getTLSConfig(client string, config httpsnapApi.Config, reload bool) (*tls.Config, errors.Error) {

	//Get cryptosuite provider name from name from peerconfig
	cryptoProvider, err := config.GetCryptoProvider()
	if err != nil {
		return nil, err
	}

	//Get cryptosuite from peer bccsp pool
	cryptoSuite, e := factory.GetBCCSP(cryptoProvider)
	if e != nil {
		return nil, errors.WithMessage(errors.CryptoConfigError, e, "failed to get crypto suite for httpsnap")
	}

	clientCert, caCerts, err := httpServiceImpl.validateClientConfig(client, config)
	if err != nil {
		return nil, err
	}

	var pk bccsp.Key

	//Get Key from Pem bytes
	key, codedErr := httpServiceImpl.getPublicKeyFromPem([]byte(clientCert), cryptoSuite)
	if codedErr != nil {
		return nil, codedErr
	}

	//Get private key using SKI
	pk, e = getKey(key.SKI(), config, cryptoProvider, reload)
	if e != nil {
		return nil, errors.Wrap(errors.GetKeyError, e, "failed to get private key from SKI")
	}

	if pk != nil && pk.Private() {
		//If private key available then get tls config from private key
		return httpServiceImpl.prepareTLSConfigFromPrivateKey(cryptoSuite, pk, clientCert, caCerts, config.IsSystemCertPoolEnabled())

	} else if config.IsPeerTLSConfigEnabled() {
		// If private key not found and allowPeerConfig enabled, then use peer tls client key
		peerClientTLSKey, err := config.GetPeerClientKey()
		if err != nil {
			return nil, errors.WithMessage(errors.MissingConfigDataError, err, "failed to get peer tls client key")
		}
		return httpServiceImpl.prepareTLSConfigFromClientKeyBytes(clientCert, peerClientTLSKey, caCerts, config.IsSystemCertPoolEnabled())

	} else {
		return nil, errors.WithMessage(errors.SystemError, err, " failed to get private key from client cert")
	}
}
func (httpServiceImpl *HTTPServiceImpl) validateClientConfig(client string, config httpsnapApi.Config) (string, []string, errors.Error) {
	var err errors.Error
	var clientCert string
	var caCerts []string

	if client != "" {
		//Use client TLS config override in https snap config
		clientOverrideCrtMap := config.GetNamedClientOverride()
		clientOverrideCrt := clientOverrideCrtMap[client]
		if clientOverrideCrt == nil {
			return clientCert, caCerts, errors.Errorf(errors.MissingConfigDataError, "client[%s] crt not found", client)
		}

		clientCert = clientOverrideCrt.Crt
		caCerts = []string{clientOverrideCrt.Ca}

	} else {

		// Use default TLS config in https snap config
		clientCert, err = config.GetClientCert()
		if err != nil {
			return clientCert, caCerts, errors.WithMessage(errors.MissingConfigDataError, err, "failed to get client cert from httpsnap config")
		}

		caCerts, err = config.GetCaCerts()
		if err != nil {
			return clientCert, caCerts, errors.WithMessage(errors.MissingConfigDataError, err, "failed to get ca certs from httpsnap config")
		}

	}
	return clientCert, caCerts, nil
}
func (httpServiceImpl *HTTPServiceImpl) validatePeerTLSClientKey(config httpsnapApi.Config) (string, errors.Error) {
	peerClientTLSKey, err := config.GetPeerClientKey()
	if err != nil {
		return peerClientTLSKey, errors.WithMessage(errors.MissingConfigDataError, err, "failed to get peer tls client key")
	}
	return peerClientTLSKey, nil
}
func (httpServiceImpl *HTTPServiceImpl) prepareTLSConfigFromClientKeyBytes(clientCert, clientKey string, caCerts []string, systemCertPoolEnabled bool) (*tls.Config, errors.Error) {
	// Load client cert
	cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		return nil, errors.Wrap(errors.CryptoError, err, "Failed to parse X509KeyPair")
	}

	return httpServiceImpl.prepareTLSConfigFromCert(cert, caCerts, systemCertPoolEnabled)
}

func (httpServiceImpl *HTTPServiceImpl) prepareTLSConfigFromCert(cert tls.Certificate, caCerts []string, systemCertPoolEnabled bool) (*tls.Config, errors.Error) {

	httpServiceImpl.certPool.Add(decodeCerts(caCerts)...)
	pool, err := httpServiceImpl.certPool.Get()
	if err != nil {
		return nil, errors.Wrap(errors.SystemError, err, "failed to create cert pool")
	}

	verifyPeerCerts := func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		if err := verifier.VerifyPeerCertificate(rawCerts, verifiedChains); err != nil {
			errObj := errors.WithMessage(errors.AccessDeniedError, err, "VerifyPeerCertificate failed")
			logger.Error(errObj.GenerateLogMsg())
			return err
		}
		return nil
	}

	// Setup HTTPS client
	return &tls.Config{
		Certificates:          []tls.Certificate{cert},
		RootCAs:               pool,
		VerifyPeerCertificate: verifyPeerCerts,
	}, nil
}
func (httpServiceImpl *HTTPServiceImpl) prepareTLSConfigFromPrivateKey(bccspSuite bccsp.BCCSP, clientKey bccsp.Key, clientCert string, caCerts []string, systemCertPoolEnabled bool) (*tls.Config, errors.Error) {
	// Load client cert
	cert, codedErr := x509KeyPair([]byte(clientCert), clientKey, bccspSuite)
	if codedErr != nil {
		return nil, codedErr
	}

	return httpServiceImpl.prepareTLSConfigFromCert(cert, caCerts, systemCertPoolEnabled)
}

func x509KeyPair(certPEMBlock []byte, clientKey bccsp.Key, bccspSuite bccsp.BCCSP) (tls.Certificate, errors.Error) {

	fail := func(err errors.Error) (tls.Certificate, errors.Error) { return tls.Certificate{}, err }

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
		return fail(errors.Wrap(errors.ParseCertError, err, "Failed x509.ParseCertificate"))
	}

	switch x509Cert.PublicKey.(type) {
	case *rsa.PublicKey:
		cert.PrivateKey = &PrivateKey{bccspSuite, clientKey, &rsa.PublicKey{}}
	case *ecdsa.PublicKey:
		cert.PrivateKey = &PrivateKey{bccspSuite, clientKey, &ecdsa.PublicKey{}}
	default:
		return fail(errors.New(errors.CryptoError, "tls: unknown public key algorithm"))
	}

	return cert, nil
}

func (httpServiceImpl *HTTPServiceImpl) validate(contentType string, schema string, body string) errors.Error {

	switch contentType {
	case "application/json":
		return httpServiceImpl.validateJSON(schema, body)
	default:
		return errors.Errorf(errors.ValidationError, "Unsupported content type: '%s' ", contentType)
	}
}

func (httpServiceImpl *HTTPServiceImpl) validateJSON(jsonSchema string, jsonStr string) errors.Error {
	logger.Debugf("Validating %s against schema: %s", jsonStr, jsonSchema)

	schemaLoader := gojsonschema.NewStringLoader(jsonSchema)
	result, err := gojsonschema.Validate(schemaLoader, gojsonschema.NewStringLoader(jsonStr))
	if err != nil {
		return errors.Wrap(errors.ValidationError, err, "Failed gojsonschema.Validate")
	}

	if !result.Valid() {
		errMsg := ""
		for i, desc := range result.Errors() {
			errMsg += desc.Description()
			if i+1 < len(result.Errors()) {
				errMsg += ", "
			}
		}
		return errors.New(errors.ValidationError, errMsg)

	}
	return nil
}

func (httpServiceImpl *HTTPServiceImpl) getPublicKeyFromPem(idBytes []byte, cryptoSuite bccsp.BCCSP) (bccsp.Key, errors.Error) {
	if len(idBytes) == 0 {
		return nil, errors.New(errors.MissingConfigDataError, "getPublicKeyFromPem error: empty pem bytes")
	}

	// Decode the pem bytes
	pemCert, _ := pem.Decode(idBytes)
	if pemCert == nil {
		return nil, errors.Errorf(errors.ParseCertError, "getPublicKeyFromPem error: could not decode pem bytes [%v]", idBytes)
	}

	// get a cert from pem bytes
	cert, err := x509.ParseCertificate(pemCert.Bytes)
	if err != nil {
		return nil, errors.Wrap(errors.ParseCertError, err, "getPublicKeyFromPem error: failed to parse x509 cert")
	}
	// get the public key in the right format
	certPubK, err := cryptoSuite.KeyImport(cert, &bccsp.X509PublicKeyImportOpts{Temporary: true})
	if err != nil {
		return nil, errors.Wrap(errors.ImportKeyError, err, "getPublicKeyFromPem error: Failed to import public key")
	}

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
