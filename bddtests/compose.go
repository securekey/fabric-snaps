/*
Copyright IBM Corp. 2016 All Rights Reserved.

SPDX-License-Identifier: Apache-2.0

*/

package bddtests

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
)

const dockerComposeCommand = "docker-compose"

// Composition represents a docker-compose execution and management
type Composition struct {
	endpoint      string
	dockerClient  *docker.Client
	apiContainers []docker.APIContainers

	composeFilesYaml string
	projectName      string
	dockerHelper     DockerHelper
}

// NewComposition create a new Composition specifying the project name (for isolation) and the compose files.
func NewComposition(projectName string, composeFilesYaml string, dir string) (composition *Composition, err error) {
	errRetFunc := func() error {
		return fmt.Errorf("Error creating new composition '%s' using compose yaml '%s':  %s", projectName, composeFilesYaml, err)
	}

	endpoint := "unix:///var/run/docker.sock"
	composition = &Composition{composeFilesYaml: composeFilesYaml, projectName: projectName}
	if composition.dockerClient, err = docker.NewClient(endpoint); err != nil {
		return nil, errRetFunc()
	}
	if _, err = composition.issueCommand([]string{"up", "--force-recreate", "-d"}, dir); err != nil {
		return nil, errRetFunc()
	}
	if composition.dockerHelper, err = NewDockerCmdlineHelper(); err != nil {
		return nil, errRetFunc()
	}
	// Now parse the current system
	return composition, nil
}

func parseComposeFilesArg(composeFileArgs string) []string {
	var args []string
	for _, f := range strings.Fields(composeFileArgs) {
		args = append(args, []string{"-f", f}...)
	}
	return args
}

func (c *Composition) getFileArgs() []string {
	return parseComposeFilesArg(c.composeFilesYaml)
}

// GetContainerIDs returns the container IDs for the composition (NOTE: does NOT include those defined outside composition, eg. chaincode containers)
func (c *Composition) GetContainerIDs(dir string) (containerIDs []string, err error) {
	var cmdOutput []byte
	if cmdOutput, err = c.issueCommand([]string{"ps", "-q"}, dir); err != nil {
		return nil, fmt.Errorf("Error getting container IDs for project '%s':  %s", c.projectName, err)
	}
	containerIDs = splitDockerCommandResults(string(cmdOutput))
	return containerIDs, err
}

func (c *Composition) refreshContainerList() (err error) {
	var allAPIContainers []docker.APIContainers
	var thisProjectsContainers []docker.APIContainers
	if thisProjectsContainers, err = c.dockerClient.ListContainers(docker.ListContainersOptions{All: true, Filters: map[string][]string{"name": {c.projectName}}}); err != nil {
		return fmt.Errorf("Error refreshing container list for project '%s':  %s", c.projectName, err)
	}
	//if allApiContainers, err = c.dockerClient.ListContainers(docker.ListContainersOptions{All: true}); err != nil {
	//	return fmt.Errorf("Error refreshing container list for project '%s':  %s", c.projectName, err)
	//}
	for _, apiContainer := range allAPIContainers {
		if composeService, ok := apiContainer.Labels["com.docker.compose.service"]; ok == true {
			fmt.Println(fmt.Sprintf("Container name:  %s, composeService: %s, IPAddress: %s", apiContainer.Names[0], composeService, apiContainer.Networks.Networks["bridge"].IPAddress))
		}
	}
	c.apiContainers = thisProjectsContainers
	return err
}

func (c *Composition) issueCommand(args []string, dir string) (_ []byte, err error) {
	var cmdOut []byte
	errRetFunc := func() error {
		return fmt.Errorf("Error issuing command to docker-compose with args '%s':  %s (%s)", args, err, string(cmdOut))
	}
	var cmdArgs []string
	cmdArgs = append(cmdArgs, c.getFileArgs()...)
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command(dockerComposeCommand, cmdArgs...)
	cmd.Dir = dir
	if cmdOut, err = cmd.CombinedOutput(); err != nil {
		return cmdOut, errRetFunc()
	}

	// Reparse Container list
	if err = c.refreshContainerList(); err != nil {
		return nil, errRetFunc()
	}
	return cmdOut, err
}

// Decompose decompose the composition.  Will also remove any containers with the same projectName prefix (eg. chaincode containers)
func (c *Composition) Decompose(dir string) (output string, err error) {
	var outputBytes []byte
	//var containers []string
	outputBytes, err = c.issueCommand([]string{"stop"}, dir)
	outputBytes, err = c.issueCommand([]string{"rm", "-f"}, dir)
	// Now remove associated chaincode containers if any
	c.dockerHelper.RemoveContainersWithNamePrefix(c.projectName)
	return string(outputBytes), err
}

// GenerateLogs to file
func (c *Composition) GenerateLogs(dir string) error {
	outputBytes, err := c.issueCommand([]string{"logs"}, dir)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("docker-compose.log", outputBytes, 775)
	return err
}

// parseComposition parses the current docker-compose project from ps command
func (c *Composition) parseComposition() (err error) {
	//c.issueCommand()
	return nil
}

// GetAPIContainerForComposeService return the docker.APIContainers with the supplied composeService name.
func (c *Composition) GetAPIContainerForComposeService(composeService string) (apiContainer *docker.APIContainers, err error) {
	for _, apiContainer := range c.apiContainers {
		if currComposeService, ok := apiContainer.Labels["com.docker.compose.service"]; ok == true {
			if currComposeService == composeService {
				return &apiContainer, nil
			}
		}
	}
	return nil, fmt.Errorf("Could not find container with compose service '%s'", composeService)
}

// GetIPAddressForComposeService returns the IPAddress of the container with the supplied composeService name.
func (c *Composition) GetIPAddressForComposeService(composeService string) (ipAddress string, err error) {
	errRetFunc := func() error {
		return fmt.Errorf("Error getting IPAddress for compose service '%s':  %s", composeService, err)
	}
	var apiContainer *docker.APIContainers
	if apiContainer, err = c.GetAPIContainerForComposeService(composeService); err != nil {
		return "", errRetFunc()
	}
	// Now get the IPAddress
	return apiContainer.Networks.Networks["bridge"].IPAddress, nil
}
