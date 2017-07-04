/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package snapdispatcher

import (
	"os"

	"os/signal"
	"syscall"
	"time"

	"fmt"

	"github.com/securekey/fabric-snaps/api/config"
	"github.com/securekey/fabric-snaps/pkg/snapdispatcher/registry"
)

// SnapsDaemon runs the Snaps Dispatcher and the provided local Snaps
type SnapsDaemon interface {
	// Initialize initializes the Snaps daemon
	// - localSnaps - an array of snaps which are running locally
	Initialize(localSnaps []*config.SnapConfig) error

	// Start starts the daemon
	Start() error
}

type snapsDaemon struct {
	registry registry.SnapsRegistry
}

// NewSnapsDaemon returns a new snaps daemon
func NewSnapsDaemon() SnapsDaemon {
	return &snapsDaemon{}
}

func (d *snapsDaemon) Initialize(localSnaps []*config.SnapConfig) error {
	registry := registry.NewSnapsRegistry(localSnaps)
	if err := registry.Initialize(); err != nil {
		return fmt.Errorf("error initializing Snap registry: %s", err)
	}

	d.registry = registry
	return nil
}

func (d *snapsDaemon) Start() error {
	logger.Info("***** Snaps daemon is starting *****")

	//start Snap server
	snapServerError := make(chan error)

	go func() {
		err := startSnapServer(d.registry)
		if err != nil {
			snapServerError <- err
		}
		snapServerError <- nil
	}()

	select {
	case err := <-snapServerError:
		if err != nil {
			logger.Errorf("Error Starting Snap Server: %s.", err)
			return err
		}
		logger.Info("Snap Server Started successfully.")

	case <-time.After(15 * time.Second):
		logger.Error("Timed out from Start Snap Server")

	}

	done := make(chan error)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-signals
		logger.Infof("Got signal: %s \n", sig)
		logger.Info("Snaps daemon is exiting")
		done <- nil
	}()

	<-done
	return nil
}
