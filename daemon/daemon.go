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
)

var logger = logging.MustGetLogger("snap-daemon")

func main() {
	err := config.Init("")
	if err != nil {
		logger.Debug("Error from config Init %+v \n", err)
		logger.Debug("SnapConfigs daemon will not start")
		os.Exit(2)
	} else {
		logger.Info("Snap Daemon configs are now loaded.")
	}



	done := make(chan error)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-signals
		logger.Infof("Got signal: %+v \n", sig)
		logger.Info("Snaps daemon is exiting")
		done <- nil
	}()

	<-done
	return
}