/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/DATA-DOG/godog"
	"github.com/spf13/viper"
)

// HTTPServerSteps ...
type HTTPServerSteps struct {
	BDDContext *BDDContext
}

// NewHTTPServerSteps ...
func NewHTTPServerSteps(context *BDDContext) *HTTPServerSteps {
	return &HTTPServerSteps{BDDContext: context}
}

func (d *HTTPServerSteps) startHTTPServer() error {

	go startTestHTTPServer()

	return nil
}

func startTestHTTPServer() {

	initHTTPServerConfig()

	// Register request handlers
	http.HandleFunc("/hello", HelloServer)

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

func initHTTPServerConfig() {
	viper.Set("http.listen.address", ":8443")
	viper.Set("http.tls.caCert.file", "./fixtures/httpserver/test-client.crt")
	viper.Set("http.tls.cert.file", "./fixtures/httpserver/server.crt")
	viper.Set("http.tls.key.file", "./fixtures/httpserver/server.key")
}

func (d *HTTPServerSteps) registerSteps(s *godog.Suite) {
	s.BeforeScenario(d.BDDContext.beforeScenario)
	s.AfterScenario(d.BDDContext.afterScenario)
	s.Step("^HTTPS Server has been started", d.startHTTPServer)
}
