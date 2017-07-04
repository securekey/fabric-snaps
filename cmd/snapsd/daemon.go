/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"os"

	"fmt"

	"github.com/op/go-logging"
	"github.com/securekey/fabric-snaps/api/config"
	"github.com/securekey/fabric-snaps/pkg/snapdispatcher"
	"github.com/securekey/fabric-snaps/pkg/snaps/examplesnap"
	"github.com/securekey/fabric-snaps/pkg/snaps/httpsnap"
)

var logger = logging.MustGetLogger("snap-snapsd")

// snaps contains an array of local Snap implementations for this Snaps container
var snaps = []*config.SnapConfig{
	// Example
	{
		Name: "examplesnap",
		Snap: &examplesnap.ExampleSnap{},
	},
	{
		Name: "httpsnap",
		Snap: &httpsnap.CCSnapImpl{},
	},
}

func main() {
	fmt.Println("***** Daemon is getting call, in snapsd *****")
	snapsDaemon := snapdispatcher.NewSnapsDaemon()

	if err := snapsDaemon.Initialize(snaps); err != nil {
		logger.Errorf("Error initializing Snap Daemon: %s\n", err)
		os.Exit(2)
	}

	if err := snapsDaemon.Start(); err != nil {
		logger.Errorf("Error starting Snap Daemon: %s\n", err)
	}
}
