/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"fmt"
	"math/rand"
	"time"

	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	fabricCommon "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"

	"github.com/spf13/viper"
)

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

// HasPrimaryPeerJoinedChannel checks whether the primary peer of a channel
// has already joined the channel. It returns true if it has, false otherwise,
// or an error
func HasPrimaryPeerJoinedChannel(channelID string, client sdkApi.Resource, orgUser sdkApi.IdentityContext, peer sdkApi.Peer) (bool, error) {
	foundChannel := false

	response, err := client.QueryChannels(peer)
	if err != nil {
		return false, fmt.Errorf("Error querying channel for primary peer: %s", err)
	}
	for _, responseChannel := range response.Channels {
		if responseChannel.ChannelId == channelID {
			foundChannel = true
		}
	}

	return foundChannel, nil
}

// GetByteArgs is a utility which converts []string to [][]bytes
func GetByteArgs(argsArray []string) [][]byte {
	txArgs := make([][]byte, len(argsArray))
	for i, val := range argsArray {
		txArgs[i] = []byte(val)
	}
	return txArgs
}

// NewCollectionConfig return CollectionConfig
func NewCollectionConfig(collName string, requiredPeerCount, maxPeerCount int32, policy *common.SignaturePolicyEnvelope) *common.CollectionConfig {
	return &common.CollectionConfig{
		Payload: &common.CollectionConfig_StaticCollectionConfig{
			StaticCollectionConfig: &common.StaticCollectionConfig{
				Name:              collName,
				RequiredPeerCount: requiredPeerCount,
				MaximumPeerCount:  maxPeerCount,
				MemberOrgsPolicy: &common.CollectionPolicyConfig{
					Payload: &common.CollectionPolicyConfig_SignaturePolicy{
						SignaturePolicy: policy,
					},
				},
			},
		},
	}
}

// IsChaincodeInstalled Helper function to check if chaincode has been deployed
func IsChaincodeInstalled(client sdkApi.Resource, peer sdkApi.Peer, name string) (bool, error) {
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

func processorsAsString(processors ...sdkApi.ProposalProcessor) string {
	str := ""
	for _, p := range processors {
		str += p.(sdkApi.Peer).URL() + " "
	}
	return str
}

func peersAsString(peers []sdkApi.Peer) string {
	str := ""
	for _, p := range peers {
		str += p.URL() + " "
	}
	return str
}

func newPolicy(policyString string) (*fabricCommon.SignaturePolicyEnvelope, error) {
	ccPolicy, err := cauthdsl.FromString(policyString)
	if err != nil {
		return nil, errors.Errorf("invalid chaincode policy [%s]: %s", policyString, err)
	}
	return ccPolicy, nil
}
