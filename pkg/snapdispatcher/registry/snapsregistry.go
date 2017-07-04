/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package registry

import (
	"fmt"
	"reflect"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	logging "github.com/op/go-logging"
	"github.com/securekey/fabric-snaps/api/config"
	snapapi "github.com/securekey/fabric-snaps/api/interfaces"
	"github.com/securekey/fabric-snaps/pkg/snapdispatcher/proxysnap"
	"github.com/spf13/viper"
)

var logger = logging.MustGetLogger("snaps-registry")

// SnapsRegistry maintains a collection of Snaps
type SnapsRegistry interface {
	// Initialize initializes the registry
	Initialize() error

	// GetSnap returns a Snap for the given name, or nil if the Snap is not registered
	GetSnap(snapName string) *config.SnapConfig
}

type snapRegistry struct {
	snaps []*config.SnapConfig
}

// NewSnapsRegistry creates a new Snaps registry
func NewSnapsRegistry(localSnaps []*config.SnapConfig) SnapsRegistry {
	registry := &snapRegistry{}
	registry.snaps = append(registry.snaps, localSnaps...)
	return registry
}

func (r *snapRegistry) Initialize() error {
	err := config.Init("")
	if err != nil {
		logger.Errorf("Error initializing Snap configs: %s \n", err)
		return fmt.Errorf("error initializing Snap configs: %s", err)
	}

	logger.Info("Snap configs are now loaded.")

	err = r.initializeSnapConfigs()
	if err != nil {
		logger.Criticalf("Error initializing snaps: %s", err)
		return fmt.Errorf("Error initializing snaps: %s", err)
	}

	// Initialize each snap
	for _, snap := range r.snaps {
		initializeSnap(snap)
	}

	logger.Debug("Snaps are ready to be used.", len(r.snaps), "snaps configs are added from the config.")

	return nil
}

func (r *snapRegistry) GetSnap(snapName string) *config.SnapConfig {
	for _, snap := range r.snaps {
		if snap.Name == snapName {
			logger.Debugf("Found registered snap %s", snap.Name)
			return snap
		}
	}

	return nil
}

func (r *snapRegistry) initializeSnapConfigs() error {
	snapConfig := &config.SnapConfigArray{}
	err := viper.UnmarshalKey("snaps", &snapConfig.SnapConfigs)

	if err != nil {
		return err
	}

	logger.Debug("Found", len(snapConfig.SnapConfigs), "snaps config(s) in yaml file.")

	for _, snapConfigCopy := range snapConfig.SnapConfigs {
		var snapMetaData = resolveSnapInitAndImplementation(&snapConfigCopy)
		if len(snapMetaData.SnapURL) > 0 {
			snapMetaData.Snap = proxysnap.NewSnap(snapMetaData.TLSEnabled, snapMetaData.TLSRootCertFile)

			// Prepend the URL of the remote snap as the first arg
			var proxySnapArgs [][]byte
			proxySnapArgs = append(proxySnapArgs, []byte(snapMetaData.SnapURL))
			proxySnapArgs = append(proxySnapArgs, snapMetaData.InitArgs...)
			snapMetaData.InitArgs = proxySnapArgs
		}

		localSnap := r.getLocalSnap(snapMetaData.Name)
		if localSnap != nil {
			// Apply the config from the file to the local snap
			snap := localSnap.Snap
			*localSnap = snapMetaData
			if snapMetaData.Snap == nil {
				localSnap.Snap = snap
			}
			logger.Debugf("Applied config to local snap %s: Enabled: %v, Impl=%v, URL=%s, TLSEnabled: %v, TLSRootCertFile: %s, Args: %v\n",
				localSnap.Name, localSnap.Enabled, reflect.TypeOf(localSnap.Snap), localSnap.SnapURL, localSnap.TLSEnabled, localSnap.TLSRootCertFile, localSnap.InitArgsStr)
		} else {
			r.snaps = append(r.snaps, &snapMetaData)
			logger.Debugf("Added snap %s: Enabled: %v, Impl=%v, URL=%s, TLSEnabled: %v, TLSRootCertFile: %s, Args: %v\n",
				snapMetaData.Name, snapMetaData.Enabled, reflect.TypeOf(snapMetaData.Snap), snapMetaData.SnapURL, snapMetaData.TLSEnabled, snapMetaData.TLSRootCertFile, snapMetaData.InitArgsStr)
		}
	}

	return nil
}

func (r *snapRegistry) getLocalSnap(name string) *config.SnapConfig {
	for _, snapConfig := range r.snaps {
		if snapConfig.Name == name {
			return snapConfig
		}
	}
	return nil
}

func resolveSnapInitAndImplementation(sp *config.SnapConfig) config.SnapConfig {
	for _, initArgVal := range sp.InitArgsStr {
		logger.Debugf("Appending init arg: %s, concatenating as a byte array: %s\n", initArgVal, []byte(initArgVal))
		sp.InitArgs = append(sp.InitArgs, []byte(initArgVal))
	}
	logger.Debug(len(sp.InitArgs), "InitArgs for snap", sp.Name, "configured.")

	return *sp
}

func initializeSnap(snap *config.SnapConfig) {
	logger.Infof("Initializing snap [%s]\n", snap.Name)
	if snap.Snap == nil {
		logger.Errorf("No implementation provided for snap: %s. The snap will be disabed.\n", snap.Name)
		snap.Enabled = false
		return
	}

	// The args to the Init method are the snap name followed by the args from the config file
	var args [][]byte
	args = append(args, []byte(snap.Name))
	args = append(args, snap.InitArgs...)

	response := snap.Snap.Init(snapapi.NewSnapStub(args))
	if response.Status != shim.OK {
		logger.Errorf("Error received from snap [%s] during initialization: [%d: %s]\n", snap.Name, response.Status, response.Message)
	}
}
