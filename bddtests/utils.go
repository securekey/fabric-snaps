/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"time"

	api "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	fabapi "github.com/hyperledger/fabric-sdk-go/def/fabapi"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/bccsp"
	fabricCaUtil "github.com/securekey/fabric-snaps/internal/github.com/hyperledger/fabric-ca/util"
	"github.com/spf13/viper"
)

// GetOrdererAdmin returns a pre-enrolled orderer admin user
func GetOrdererAdmin(c api.FabricClient, orgName string) (api.User, error) {
	keyDir := "ordererOrganizations/example.com/users/Admin@example.com/msp/keystore"
	certDir := "ordererOrganizations/example.com/users/Admin@example.com/msp/signcerts"
	return getDefaultImplPreEnrolledUser(c, keyDir, certDir, "ordererAdmin", orgName)
}

// GetAdmin returns a pre-enrolled org admin user
func GetAdmin(c api.FabricClient, orgPath string, orgName string) (api.User, error) {
	keyDir := fmt.Sprintf("peerOrganizations/%s.example.com/users/Admin@%s.example.com/msp/keystore", orgPath, orgPath)
	certDir := fmt.Sprintf("peerOrganizations/%s.example.com/users/Admin@%s.example.com/msp/signcerts", orgPath, orgPath)
	username := fmt.Sprintf("peer%sAdmin", orgPath)
	return getDefaultImplPreEnrolledUser(c, keyDir, certDir, username, orgName)
}

// GetUser returns a pre-enrolled org user
func GetUser(c api.FabricClient, orgPath string, orgName string) (api.User, error) {
	keyDir := fmt.Sprintf("peerOrganizations/%s.example.com/users/User1@%s.example.com/msp/keystore", orgPath, orgPath)
	certDir := fmt.Sprintf("peerOrganizations/%s.example.com/users/User1@%s.example.com/msp/signcerts", orgPath, orgPath)
	username := fmt.Sprintf("peer%sUser1", orgPath)
	return getDefaultImplPreEnrolledUser(c, keyDir, certDir, username, orgName)
}

// GetChannelTxPath returns path to the channel tx file for the given channel
func GetChannelTxPath(channelID string) string {
	return viper.GetString(fmt.Sprintf("bddtest.channelconfig.%s.txpath", channelID))
}

// GetChannelAnchorTxPath returns path to the channel anchor tx file for the given channel
func GetChannelAnchorTxPath(channelID, orgName string) string {
	return viper.GetString(fmt.Sprintf("bddtest.channelconfig.%s.anchortxpath.%s", channelID, orgName))
}

// GenerateRandomID generates random ID
func GenerateRandomID() string {
	rand.Seed(time.Now().UnixNano())
	return randomString(10)
}

// Utility to create random string of strlen length
func randomString(strlen int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

// GetDefaultImplPreEnrolledUser ...
func getDefaultImplPreEnrolledUser(client api.FabricClient, keyDir string, certDir string, username string, orgName string) (api.User, error) {

	privateKeyDir := filepath.Join(client.Config().CryptoConfigPath(), keyDir)
	privateKeyPath, err := getFirstPathFromDir(privateKeyDir)
	if err != nil {
		return nil, fmt.Errorf("Error finding the private key path: %v", err)
	}

	enrollmentCertDir := filepath.Join(client.Config().CryptoConfigPath(), certDir)
	enrollmentCertPath, err := getFirstPathFromDir(enrollmentCertDir)
	if err != nil {
		return nil, fmt.Errorf("Error finding the enrollment cert path: %v", err)
	}

	mspID, err := client.Config().MspID(orgName)
	if err != nil {
		return nil, fmt.Errorf("Error reading MSP ID config: %s", err)
	}

	signingIdentity, err := getSigningIdentity(mspID, privateKeyPath, enrollmentCertPath, client.CryptoSuite())
	if err != nil {
		return nil, fmt.Errorf("Failed to get signing identity %v", err)
	}

	return fabapi.NewPreEnrolledUser(client.Config(), username, signingIdentity)
}

// Gets the first path from the dir directory
func getFirstPathFromDir(dir string) (string, error) {

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("Could not read directory %s, err %s", err, dir)
	}

	for _, p := range files {
		if p.IsDir() {
			continue
		}

	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		fullName := filepath.Join(dir, string(filepath.Separator), f.Name())
		return fullName, nil
	}

	return "", fmt.Errorf("No paths found in directory: %s", dir)
}

// HasPrimaryPeerJoinedChannel checks whether the primary peer of a channel
// has already joined the channel. It returns true if it has, false otherwise,
// or an error
func HasPrimaryPeerJoinedChannel(client api.FabricClient, orgUser api.User, channel api.Channel) (bool, error) {
	foundChannel := false
	primaryPeer := channel.PrimaryPeer()

	currentUser := client.UserContext()
	defer client.SetUserContext(currentUser)

	client.SetUserContext(orgUser)
	response, err := client.QueryChannels(primaryPeer)
	if err != nil {
		return false, fmt.Errorf("Error querying channel for primary peer: %s", err)
	}
	for _, responseChannel := range response.Channels {
		if responseChannel.ChannelId == channel.Name() {
			foundChannel = true
		}
	}

	return foundChannel, nil
}

// IsChaincodeInstalled Helper function to check if chaincode has been deployed
func IsChaincodeInstalled(client api.FabricClient, peer api.Peer, name string) (bool, error) {
	chaincodeQueryResponse, err := client.QueryInstalledChaincodes(peer)
	if err != nil {
		return false, err
	}

	for _, chaincode := range chaincodeQueryResponse.Chaincodes {
		if chaincode.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func getFilesWithName(pathRelToWD string, fileName string) ([]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	var files []string
	filepath.Walk(wd+"/"+pathRelToWD, func(path string, f os.FileInfo, _ error) error {
		if !f.IsDir() {
			r, err := regexp.MatchString(fileName, f.Name())
			if err == nil && r {
				files = append(files, path)
			}
		}
		return nil
	})

	return files, nil
}

func getSigningIdentity(mspID string, privateKeyPath string, enrollmentCertPath string, cryptoSuite bccsp.BCCSP) (*api.SigningIdentity, error) {

	privateKey, err := fabricCaUtil.ImportBCCSPKeyFromPEM(privateKeyPath, cryptoSuite, true)
	if err != nil {
		return nil, fmt.Errorf("Error importing private key: %v", err)
	}
	enrollmentCert, err := ioutil.ReadFile(enrollmentCertPath)
	if err != nil {
		return nil, fmt.Errorf("Error reading from the enrollment cert path: %v", err)
	}

	signingIdentity := &api.SigningIdentity{MspID: mspID, PrivateKey: privateKey, EnrollmentCert: enrollmentCert}

	return signingIdentity, nil
}
