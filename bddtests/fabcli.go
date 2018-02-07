/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// FabCLI is used to invoke the Fabric CLI command-line tool
type FabCLI struct {
}

// NewFabCLI returns a new FabCLI
func NewFabCLI() *FabCLI {
	return &FabCLI{}
}

// GetJSON executes the fabric-cli with the given args and returns a JSON response
func (cli *FabCLI) GetJSON(args ...string) (string, error) {
	newArgs := append(args, "--format")
	newArgs = append(newArgs, "json")

	respStr, err := cli.Exec(newArgs...)
	if err != nil {
		return "", err
	}

	// The Go SDK adds some logging that we need to remove
	i := strings.Index(respStr, "{")
	if i < 0 {
		return "", fmt.Errorf("JSON not found in response")
	}

	return strings.Replace(respStr[i:], "\n", "", -1), nil
}

// Exec executes the fabric-cli with the given args and returns a response
func (cli *FabCLI) Exec(args ...string) (string, error) {
	cmd := exec.Command("fabric-cli", args...)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Start()
	if err != nil {
		return "", err
	}
	err = cmd.Wait()
	if err != nil {
		return "", err
	}
	return out.String(), nil
}
