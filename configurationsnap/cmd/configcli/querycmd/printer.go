/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package querycmd

import (
	"encoding/json"
	"fmt"
	"strings"

	mgmtapi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/cliconfig"
)

// OutputFormat specifies the format for printing data
type OutputFormat uint8

const (
	// RawOutput displays the raw data
	RawOutput OutputFormat = iota

	// FormattedOutput formats the data into a human readable format
	FormattedOutput
)

func (f OutputFormat) String() string {
	switch f {
	case FormattedOutput:
		return "formatted"
	case RawOutput:
		return "raw"
	default:
		return "unknown"
	}
}

// AsOutputFormat returns the OutputFormat given an Output Format string
func AsOutputFormat(f string) OutputFormat {
	switch strings.ToLower(f) {
	case "raw":
		return RawOutput
	default:
		return FormattedOutput
	}
}

const (
	lineSep = "--------------------------------------------------------------------"
)

// Print prints the given config bytes, which is a marshalled JSON array of ConfigKV
func Print(configBytes []byte) {
	if AsOutputFormat(cliconfig.Config().OutputFormat()) == RawOutput {
		fmt.Printf("\n%s\n%s\n", lineSep, configBytes)
	} else {
		var configs []mgmtapi.ConfigKV
		if err := json.Unmarshal(configBytes, &configs); err != nil {
			cliconfig.Config().Logger().Errorf("Got error while unmarshalling config: %v", err)
			return
		}
		for _, config := range configs {
			fmt.Printf("\n%s\n----- MSPID: %s, Peer: %s, App: %s:,Version: %s:\n%s\n%s\n", lineSep, config.Key.MspID, config.Key.PeerID, config.Key.AppName, config.Key.Version, config.Value, lineSep)
		}
	}
}
