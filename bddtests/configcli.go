/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"bytes"
	"fmt"
	"os/exec"
)

// ConfigCLI is used to invoke the Config CLI command-line tool
type ConfigCLI struct {
	clientconfig string
	user         string
	binaryPath   string
}

// NewConfigCLI returns a new NewConfigCLI
func NewConfigCLI(binaryPath, clientconfig, user string) *ConfigCLI {
	return &ConfigCLI{clientconfig, user, binaryPath}
}

// ExecUpdate executes config-cli update with the given args and returns a response
func (cli *ConfigCLI) ExecUpdate(channelID, mspID, org, configFile string) (string, error) {
	cmdArgs := []string{"update", "--clientconfig", cli.clientconfig, "--cid", channelID, "--mspid", mspID,
		"--user", cli.user, "--configfile", configFile, "--noprompt", "--orgid", org, "--timeout", "120000"}

	cmd := exec.Command(cli.binaryPath, cmdArgs...)

	var out bytes.Buffer
	cmd.Stdout = &out
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Start()
	if err != nil {
		return "", fmt.Errorf(err.Error() + ": " + stderr.String())
	}
	err = cmd.Wait()
	if err != nil {
		return "", fmt.Errorf(err.Error() + ": " + stderr.String())
	}
	return out.String(), nil
}

// Exec executes the config-cli action with the given args and returns a response
func (cli *ConfigCLI) Exec(action, channelID, mspID, peerID, appName, appVersion, compName, compVersion string) (string, error) {
	cmdArgs := []string{action, "--clientconfig", cli.clientconfig, "--cid", channelID, "--mspid", mspID,
		"--user", cli.user, "--peerid", peerID, "--appname", appName, "--appver", appVersion, "--componentname", compName, "--componentver", compVersion}
	if action == "delete" {
		cmdArgs = append(cmdArgs, "--noprompt")
	}

	cmd := exec.Command("../build/configcli", cmdArgs...)

	var out bytes.Buffer
	cmd.Stdout = &out
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Start()
	if err != nil {
		return "", fmt.Errorf(err.Error() + ": " + stderr.String())
	}
	err = cmd.Wait()
	if err != nil {
		return "", fmt.Errorf(err.Error() + ": " + stderr.String())
	}
	return out.String(), nil
}
