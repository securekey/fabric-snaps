/*
   Copyright SecureKey Technologies Inc.
   This file contains software code that is the intellectual property of SecureKey.
   SecureKey reserves all rights in the code and you may not use it without
	 written permission from SecureKey.
*/

package main

import (
	"errors"
	"fmt"

	"os"
	"strings"

	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/bccsp/pkcs11"
	shim "github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/spf13/viper"
)

var logger = shim.NewLogger("bootstrap-scc")

const (
	peerConfigFileName = "core"
	peerConfigPath     = "/etc/hyperledger/fabric"
	cmdRootPrefix      = "core"
)

var encryptLogging bool

// New chaincode implementation
func New() shim.Chaincode {
	return &BootstrapSnap{}
}

// BootstrapSnap implementation
type BootstrapSnap struct {
}

// Init snap
func (bootstrapSnap *BootstrapSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {

	err := bootstrapSnap.initBCCSP()
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to initialize bootstrap snap. Error : %v", err))
	}
	logger.Debug("bccsp initialized successfully")
	return shim.Success(nil)
}

// Invoke is the main entry point for invocations
func (bootstrapSnap *BootstrapSnap) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (bootstrapSnap *BootstrapSnap) initBCCSP() error {
	//peer Config
	peerConfig := viper.New()
	peerConfig.AddConfigPath(peerConfigPath)
	peerConfig.SetConfigName(peerConfigFileName)
	peerConfig.SetEnvPrefix(cmdRootPrefix)
	peerConfig.AutomaticEnv()
	peerConfig.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	err := peerConfig.ReadInConfig()
	if err != nil {
		return err
	}

	configuredProvider := peerConfig.GetString("peer.BCCSP.Default")
	if configuredProvider == "" {
		return errors.New("BCCSP Default provider not found")
	}

	level := peerConfig.GetInt(fmt.Sprintf("peer.BCCSP.%s.Security", configuredProvider))
	hashFamily := peerConfig.GetString(fmt.Sprintf("peer.BCCSP.%s.Hash", configuredProvider))
	ksPath := peerConfig.GetString(fmt.Sprintf("peer.BCCSP.%s.FileKeyStore.KeyStore", configuredProvider))

	pin := peerConfig.GetString(fmt.Sprintf("peer.BCCSP.%s.Pin", configuredProvider))
	label := peerConfig.GetString(fmt.Sprintf("peer.BCCSP.%s.Label", configuredProvider))
	lib := FindPKCS11Lib(peerConfig.GetString(fmt.Sprintf("peer.BCCSP.%s.Library", configuredProvider)))

	logger.Debug("Configured BCCSP provider '%s' \nlib: %s \npin: %s \nlabel: %s\n keystore: %s\n", configuredProvider, lib, pin, label, ksPath)

	var opts *factory.FactoryOpts
	switch configuredProvider {
	case "PKCS11":
		opts = &factory.FactoryOpts{
			ProviderName: "PKCS11",
			Pkcs11Opts: &pkcs11.PKCS11Opts{
				SecLevel:   level,
				HashFamily: hashFamily,
				Ephemeral:  false,
				Library:    lib,
				Pin:        pin,
				Label:      label,
				FileKeystore: &pkcs11.FileKeystoreOpts{
					KeyStorePath: ksPath,
				},
			},
		}
	case "SW":
		opts = &factory.FactoryOpts{
			ProviderName: "SW",
			SwOpts: &factory.SwOpts{
				HashFamily: hashFamily,
				SecLevel:   level,
				Ephemeral:  true,
			},
		}
	default:
		return errors.New("Unsupported PKCS11 provider")
	}
	return factory.InitFactories(opts)
}

//FindPKCS11Lib find lib based on configuration
func FindPKCS11Lib(configuredLib string) string {
	logger.Debugf("PKCS library configurations paths  %s ", configuredLib)
	var lib string
	if configuredLib != "" {
		possibilities := strings.Split(configuredLib, ",")
		for _, path := range possibilities {
			trimpath := strings.TrimSpace(path)
			if _, err := os.Stat(trimpath); !os.IsNotExist(err) {
				lib = trimpath
				break
			}
		}
	}
	logger.Debugf("Found pkcs library '%s'", lib)
	return lib
}

func main() {
}
