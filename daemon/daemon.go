/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"os"

	"github.com/securekey/fabric-snaps/config"
	"github.com/op/go-logging"
	"os/signal"
	"syscall"
	"time"
	"github.com/securekey/fabric-snaps/snaps/snapdispatcher"
)

var logger = logging.MustGetLogger("snap-daemon")

func main() {
	err := config.Init("")
	if err != nil {
		logger.Errorf("Error initializing Snap configs: %s \n", err)
		logger.Error("Snap Configs daemon will not start")
		os.Exit(2)
	} else {
		logger.Info("Snap configs are now loaded.")
	}

	//start Snap server
	SnapServerError := make(chan error)

	go func() {
		err := snapdispatcher.StartSnapServer()
		if err != nil {
			SnapServerError <- err
		}
		SnapServerError <- nil
	}()

	select {
	case err := <-SnapServerError:
		if err != nil {
			logger.Errorf("Error Starting Snap Server: %s.", err)
		} else {
			logger.Info ("Snap Server Started successfully.")
		}

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
	return
}