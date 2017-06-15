/* Copyright SecureKey Technologies Inc.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.*/

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