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
	"strings"

	"encoding/pem"

	"crypto"
	"io"

	"crypto/rsa"

	"crypto/ecdsa"

	"sync"

	"sync/atomic"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	commtls "github.com/hyperledger/fabric-sdk-go/pkg/core/config/comm/tls"
	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazycache"
	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazyref"
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

var cache *lazycache.Cache

//HTTPServiceImpl used to create transaction service
type HTTPServiceImpl struct {
	sync.RWMutex
	config     httpsnapApi.Config
	configHash atomic.Value
	certPool   fab.CertPool
}

//HTTPServiceInvokeRequest used to create http invoke service
type HTTPServiceInvokeRequest struct {
	RequestURL     string
	RequestHeaders map[string]string
	RequestBody    string
	NamedClient    string
	PinSet         []string
}

const (
	contentType = "content-type"
)

// Dialer is custom dialer to verify cert against pinset
type Dialer func(network, addr string) (net.Conn, error)

//Get will return httpService to caller
func Get(channelID string) (*HTTPServiceImpl, error) {
	return newHTTPService(channelID)
}

//updateConfig http service updates http service config if provided config has any updates
func (httpServiceImpl *HTTPServiceImpl) init(config httpsnapApi.Config) {

	if httpServiceImpl.configHash.Load() == config.GetConfigHash() {
		//no updates in config, do nothing
		return
	}

	//Update config in httpServiceImpl if any config update found in new config
	httpServiceImpl.Lock()
	defer httpServiceImpl.Unlock()

	httpServiceImpl.config = config
	httpServiceImpl.configHash.Store(config.GetConfigHash())

	httpServiceImpl.certPool = commtls.NewCertPool(config.IsSystemCertPoolEnabled())

	level, err := config.GetLogLevel()
	if err != nil {
		//only error scenario when invalid non-empty log level is provided in config file which cannot be parsed to valid SDK log level
		logger.Errorf("Unable to set log level, %s", err)
	} else {
		logging.SetLevel("httpsnap", level)
	}

}

//Invoke http service
func (httpServiceImpl *HTTPServiceImpl) Invoke(httpServiceInvokeRequest HTTPServiceInvokeRequest) ([]byte, error) {
	if httpServiceInvokeRequest.RequestURL == "" {
		return nil, errors.New(errors.GeneralError, "Missing RequestURL")
	}

	if len(httpServiceInvokeRequest.RequestHeaders) == 0 {
		return nil, errors.New(errors.GeneralError, "Missing request headers")
	}

	headers := make(map[string]string)

	// Converting header names to lowercase
	for name, value := range httpServiceInvokeRequest.RequestHeaders {
		headers[strings.ToLower(name)] = value
	}

	if _, ok := headers[contentType]; !ok {
		return nil, errors.New(errors.GeneralError, "Missing required content-type header")
	}

	if val, ok := headers[contentType]; ok && val == "" {
		return nil, errors.New(errors.GeneralError, "content-type header is empty")
	}

	httpServiceInvokeRequest.RequestHeaders = headers

	if httpServiceInvokeRequest.RequestBody == "" {
		return nil, errors.New(errors.GeneralError, "Missing RequestBody")
	}

	// Validate URL
	uri, err := url.ParseRequestURI(httpServiceInvokeRequest.RequestURL)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "Invalid URL")
	}

	// Security controls should be added by the chaincode that calls the HTTP snap

	// Scheme has to be https
	if uri.Scheme != "https" {
		return nil, errors.Errorf(errors.GeneralError, "Unsupported scheme: %s", uri.Scheme)
	}

	httpServiceImpl.RLock()
	defer httpServiceImpl.RUnlock()

	schemaConfig, err := httpServiceImpl.config.GetSchemaConfig(headers[contentType])
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "GetSchemaConfig return error")
	}

	// Validate request body against schema
	if err := httpServiceImpl.validate(headers[contentType], schemaConfig.Request, httpServiceInvokeRequest.RequestBody); err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Failed to validate request body")
	}

	// URL is ok, retrieve data using http client
	_, response, err := httpServiceImpl.getData(httpServiceInvokeRequest)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "getData return error")
	}

	logger.Debugf("Successfully retrieved data from URL: %s", httpServiceInvokeRequest.RequestURL)

	// Validate response body against schema
	if err := httpServiceImpl.validate(headers[contentType], schemaConfig.Response, string(response)); err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "validate return error")
	}
	return response, nil
}

//newHTTPService creates new http snap service
func newHTTPService(channelID string) (*HTTPServiceImpl, error) {
	config, err := httpsnapconfig.NewConfig(PeerConfigPath, channelID)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "Failed to initialize config")
	}

	if config == nil {
		return nil, errors.New(errors.GeneralError, "config from ledger is nil")
	}
	return getInstance(newCacheKey(channelID, config))
}

func getInstance(key CacheKey) (*HTTPServiceImpl, error) {
	once.Do(func() {
		logger.Debugf("Setting client cache refresh interval %d\n", key.HttpSnapConfig().GetClientCacheRefreshInterval())
		cache = newRefCache(key.HttpSnapConfig().GetClientCacheRefreshInterval())
		logger.Debug("Cache was intialized")
	})

	ref, err := cache.Get(key)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Got error while getting item from cache")
	}

	clientRef := ref.(*lazyref.Reference)
	client, err := clientRef.Get()
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "error getting client")
	}
	return client.(*HTTPServiceImpl), nil
}

func (httpServiceImpl *HTTPServiceImpl) getData(invokeReq HTTPServiceInvokeRequest) (responseContentType string, responseBody []byte, err error) {

	tlsConfig, err := httpServiceImpl.getTLSConfig(invokeReq.NamedClient, httpServiceImpl.config)
	if err != nil {
		logger.Errorf("Failed to load tls config. namedClient=%s, err=%s", invokeReq.NamedClient, err)
		return "", nil, err
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

	logger.Debugf("Requesting %s from url=%s", invokeReq.RequestBody, invokeReq.RequestURL)

	req, err := http.NewRequest("POST", invokeReq.RequestURL, bytes.NewBuffer([]byte(invokeReq.RequestBody)))
	if err != nil {
		return "", nil, errors.Wrap(errors.GeneralError, err, "Failed http.NewRequest")
	}

	// Set allowed headers only
	for name, value := range invokeReq.RequestHeaders {
		allowed, err := httpServiceImpl.config.IsHeaderAllowed(name)
		if err != nil {
			return "", nil, err
		}
		if allowed {
			req.Header.Set(name, value)
			logger.Debugf("Setting header '%s' to '%s'", name, value)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("POST failed. url=%s, err=%s", invokeReq.RequestURL, err)
		return "", nil, errors.Wrapf(errors.GeneralError, err, "POST failed. url=%s", invokeReq.RequestURL)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, errors.Errorf(errors.GeneralError, "Http response status code: %d, status: %s, url=%s", resp.StatusCode, resp.Status, invokeReq.RequestURL)
	}

	responseContentType = resp.Header.Get(contentType)

	if !strings.Contains(strings.ToLower(responseContentType), strings.ToLower(invokeReq.RequestHeaders[contentType])) {
		return "", nil, errors.Errorf(errors.GeneralError, "Response content-type: %s doesn't match request content-type: %s", responseContentType, invokeReq.RequestHeaders[contentType])
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Warnf("Read contents failed. url=%s, err=%s", invokeReq.RequestURL, err)
		return "", nil, errors.Wrapf(errors.GeneralError, err, "Read contents failed. url=%s", invokeReq.RequestURL)
	}

	logger.Debugf("Got %s from url=%s", contents, invokeReq.RequestURL)

	return responseContentType, contents, nil
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
			return nil, errors.Wrap(errors.GeneralError, err, "Failed tls.DialWithDialer")
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
			return nil, errors.Errorf(errors.GeneralError, "Failed to validate peer cert pins %s against allowed pins: %s", peerPins, pins)
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

	//Get cryptosuite provider name from name from peerconfig
	cryptoProvider, err := config.GetCryptoProvider()
	if err != nil {
		return nil, err
	}

	//Get cryptosuite from peer bccsp pool
	cryptoSuite, err := factory.GetBCCSP(cryptoProvider)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "failed to get cryptoSuite for httpsnap")
	}

	var clientCert string
	var caCerts []string
	var pk bccsp.Key

	if client != "" {
		//Use client TLS config override in https snap config
		clientOverrideCrtMap, err := config.GetNamedClientOverride()
		if err != nil {
			return nil, err
		}
		clientOverrideCrt := clientOverrideCrtMap[client]
		if clientOverrideCrt == nil {
			return nil, errors.Errorf(errors.GeneralError, "client[%s] crt not found", client)
		}

		clientCert = clientOverrideCrt.Crt
		caCerts = []string{clientOverrideCrt.Ca}

	} else {

		// Use default TLS config in https snap config
		clientCert, err = config.GetClientCert()
		if err != nil {
			return nil, errors.WithMessage(errors.GeneralError, err, "failed to get client cert from httpsnap config")
		}

		caCerts, err = config.GetCaCerts()
		if err != nil {
			return nil, errors.WithMessage(errors.GeneralError, err, "failed to get ca certs from httpsnap config")
		}

	}

	//Get Key from Pem bytes
	key, err := httpServiceImpl.getCryptoSuiteKeyFromPem([]byte(clientCert), cryptoSuite)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "failed to get key from client cert")
	}

	//Get private key using SKI
	pk, err = cryptoSuite.GetKey(key.SKI())

	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "failed to get private key from SKI")
	}

	if pk != nil && pk.Private() {
		//If private key available then get tls config from private key
		return httpServiceImpl.prepareTLSConfigFromPrivateKey(cryptoSuite, pk, clientCert, caCerts, config.IsSystemCertPoolEnabled())

	} else if config.IsPeerTLSConfigEnabled() {
		// If private key not found and allowPeerConfig enabled, then use peer tls client key
		peerClientTLSKey, err := config.GetPeerClientKey()
		if err != nil {
			return nil, errors.WithMessage(errors.GeneralError, err, "failed to get peer tls client key")
		}
		return httpServiceImpl.prepareTLSConfigFromClientKeyBytes(clientCert, peerClientTLSKey, caCerts, config.IsSystemCertPoolEnabled())

	} else {
		return nil, errors.WithMessage(errors.GeneralError, err, " failed to get private key from client cert")
	}
}

func (httpServiceImpl *HTTPServiceImpl) prepareTLSConfigFromClientKeyBytes(clientCert, clientKey string, caCerts []string, systemCertPoolEnabled bool) (*tls.Config, error) {
	// Load client cert
	cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "Failed X509KeyPair")
	}

	pool, err := httpServiceImpl.certPool.Get(decodeCerts(caCerts)...)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "failed to create cert pool")
	}

	// Setup HTTPS client
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
	}, nil
}

func (httpServiceImpl *HTTPServiceImpl) prepareTLSConfigFromPrivateKey(bccspSuite bccsp.BCCSP, clientKey bccsp.Key, clientCert string, caCerts []string, systemCertPoolEnabled bool) (*tls.Config, error) {
	// Load client cert
	tlscert, err := x509KeyPair([]byte(clientCert), clientKey, bccspSuite)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "Failed X509KeyPair")
	}

	pool, err := httpServiceImpl.certPool.Get(decodeCerts(caCerts)...)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "failed to create cert pool")
	}

	// Setup HTTPS client
	return &tls.Config{
		Certificates: []tls.Certificate{tlscert},
		RootCAs:      pool,
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
		return fail(errors.Wrap(errors.GeneralError, err, "Failed x509.ParseCertificate"))
	}

	switch x509Cert.PublicKey.(type) {
	case *rsa.PublicKey:
		cert.PrivateKey = &PrivateKey{bccspSuite, clientKey, &rsa.PublicKey{}}
	case *ecdsa.PublicKey:
		cert.PrivateKey = &PrivateKey{bccspSuite, clientKey, &ecdsa.PublicKey{}}
	default:
		return fail(errors.New(errors.GeneralError, "tls: unknown public key algorithm"))
	}

	return cert, nil
}

func (httpServiceImpl *HTTPServiceImpl) validate(contentType string, schema string, body string) error {

	switch contentType {
	case "application/json":
		return httpServiceImpl.validateJSON(schema, body)
	default:
		return errors.Errorf(errors.GeneralError, "Unsupported content type: '%s' ", contentType)
	}
}

func (httpServiceImpl *HTTPServiceImpl) validateJSON(jsonSchema string, jsonStr string) error {
	logger.Debugf("Validating %s against schema: %s", jsonStr, jsonSchema)

	schemaLoader := gojsonschema.NewStringLoader(jsonSchema)
	result, err := gojsonschema.Validate(schemaLoader, gojsonschema.NewStringLoader(jsonStr))
	if err != nil {
		return errors.Wrap(errors.GeneralError, err, "Failed gojsonschema.Validate")
	}

	if !result.Valid() {
		errMsg := ""
		for i, desc := range result.Errors() {
			errMsg += desc.Description()
			if i+1 < len(result.Errors()) {
				errMsg += ", "
			}
		}
		return errors.New(errors.GeneralError, errMsg)

	}
	return nil
}

func (httpServiceImpl *HTTPServiceImpl) getCryptoSuiteKeyFromPem(idBytes []byte, cryptoSuite bccsp.BCCSP) (bccsp.Key, error) {
	if idBytes == nil {
		return nil, errors.New(errors.GeneralError, "getCryptoSuiteKeyFromPem error: nil idBytes")
	}

	// Decode the pem bytes
	pemCert, _ := pem.Decode(idBytes)
	if pemCert == nil {
		return nil, errors.Errorf(errors.GeneralError, "getCryptoSuiteKeyFromPem error: could not decode pem bytes [%v]", idBytes)
	}

	// get a cert
	cert, err := x509.ParseCertificate(pemCert.Bytes)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "getCryptoSuiteKeyFromPem error: failed to parse x509 cert")
	}

	// get the public key in the right format
	certPubK, err := cryptoSuite.KeyImport(cert, &bccsp.X509PublicKeyImportOpts{Temporary: true})
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "Failed KeyImport")
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
