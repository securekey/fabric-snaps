/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"os"

	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/cliconfig"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/deletecmd"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/generateCSR"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/querycmd"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/updatecmd"
	"github.com/spf13/cobra"
)

func newConfigCLICmd() *cobra.Command {
	mainCmd := &cobra.Command{
		Use: "configcli",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}

	flags := mainCmd.PersistentFlags()

	cliconfig.InitLoggingLevel(flags)
	cliconfig.InitClientConfigFile(flags)
	cliconfig.InitChannelID(flags)
	cliconfig.InitUserName(flags)
	cliconfig.InitUserPassword(flags)
	cliconfig.InitOrgID(flags)
	cliconfig.InitMspID(flags)

	mainCmd.AddCommand(querycmd.Cmd(), updatecmd.Cmd(), deletecmd.Cmd(), generateCSR.Cmd())

	return mainCmd
}

func main() {
	if newConfigCLICmd().Execute() != nil {
		os.Exit(1)
	}
}
