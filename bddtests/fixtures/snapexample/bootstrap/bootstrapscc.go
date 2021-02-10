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

	"strings"

	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/bccsp/pkcs11"
	"github.com/hyperledger/fabric/common/viperutil"
	shim "github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/metrics/pkg/util"
	"github.com/spf13/viper"
)

var logger = shim.NewLogger("bootstrap-scc")

const (
	peerConfigFileName = "core"
	peerConfigPath     = "/etc/hyperledger/fabric"
	cmdRootPrefix      = "core"
	envLabel           = "CORE_PEER_BCCSP_PKCS11_LABEL"
	envPin             = "CORE_PEER_BCCSP_PKCS11_PIN"
	envLib             = "CORE_PEER_BCCSP_PKCS11_LIBRARY"
)

var encryptLogging bool

// New chaincode implementation
func New() shim.Chaincode {
	if err := util.InitializeMetricsProvider(""); err != nil {
		panic(err)
	}
	return &BootstrapSnap{}
}

// BootstrapSnap implementation
type BootstrapSnap struct {
}

// Init snap
func (bootstrapSnap *BootstrapSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Printf("################## Bootstrap CC ###############")
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
	//get peer BCCSP config
	var bccspConfig *factory.FactoryOpts
	err = viperutil.EnhancedExactUnmarshalKey("peer.BCCSP", &bccspConfig)
	if err != nil {
		return errors.New(err.Error())
	}
	logger.Debugf("BCCSP config from unmarshaller-Provider %v ", bccspConfig.ProviderName)
	logger.Debugf("BCCSP Lib %v", bccspConfig.Pkcs11Opts.Library)
	logger.Debugf("BCCSP Pin %v", bccspConfig.Pkcs11Opts.Pin)
	logger.Debugf("BCCSP Label %v", bccspConfig.Pkcs11Opts.Label)
	configuredProvider := bccspConfig.ProviderName
	level := bccspConfig.Pkcs11Opts.SecLevel
	hashFamily := bccspConfig.Pkcs11Opts.HashFamily
	ksPath := bccspConfig.Pkcs11Opts.FileKeystore.KeyStorePath
	pin := bccspConfig.Pkcs11Opts.Pin
	label := bccspConfig.Pkcs11Opts.Label
	lib := bccspConfig.Pkcs11Opts.Library

	logger.Debug("Bootstrap Configured BCCSP provider '%s' \nlib: %s \npin: %s \nlabel: %s\n keystore: %s\n", configuredProvider, lib, pin, label, ksPath)

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
			},
		}
	default:
		return errors.New("Unsupported PKCS11 provider")
	}
	return factory.InitFactories(opts)
}

func main() {
}
