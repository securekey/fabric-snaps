/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/viper"
)

func main() {
	fmt.Println("Starting go routine...")
	go startTestHTTPServer()
	fmt.Println("Entering for loop...")
	for {
	}
}

func startTestHTTPServer() {

	initHTTPServerConfig()

	fmt.Println("Registering request handlers...")
	// Register request handlers
	http.HandleFunc("/hello", HelloServer)

	fmt.Println("Reading caCert file...")
	caCert, err := ioutil.ReadFile(viper.GetString("http.tls.caCert.file"))
	if err != nil {
		fmt.Println("HTTP Server: Failed to read ca-cert. " + err.Error())
		os.Exit(1)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	fmt.Println("Setup HTTPS client...")
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

	fmt.Println("Calling ListenAndServeTLS...")
	err = server.ListenAndServeTLS(viper.GetString("http.tls.cert.file"), viper.GetString("http.tls.key.file"))

	if err != nil {
		fmt.Println("HTTP Server: Failed to start. " + err.Error())
		os.Exit(1)
	}

}

// HelloServer greeting (JSON)
func HelloServer(w http.ResponseWriter, req *http.Request) {
	fmt.Println("Calling HelloServer...")
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"description": "Hello"}`)
}

func initHTTPServerConfig() {
	replacer := strings.NewReplacer(".", "_")
	fmt.Println("Setting viper config vars...")
	viper.AddConfigPath("./")
	viper.AddConfigPath(os.Getenv("EXT_SERVER_CFG_PATH"))
	viper.AddConfigPath("/etc/external-http-server/")
	viper.SetConfigName("config")
	viper.SetEnvPrefix("core")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(replacer)

	fmt.Println("Reading in config via viper...")
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Printf("Fatal error reading config file: %s \n", err)
		os.Exit(1)
	}
}
