/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	mspApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
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

// Resolve resolves all variables within the given arg
//
// Example 1: Simple variable
// 	Given:
// 		vars = {
// 			"var1": "value1",
// 			"var2": "value2",
// 			}
//	Then:
//		"${var1}" = "value1"
//		"X_${var1}_${var2} = "X_value1_value2
//
// Example 2: Array variable
// 	Given:
// 		vars = {
// 			"arr1": "value1,value2,value3",
// 			}
//	Then:
//		"${arr1[0]_arr1[1]_arr1[2]}" = "value1_value2_value3"
//
func Resolve(vars map[string]string, arg string) (string, error) {
	for {
		str, err := doResolve(vars, arg)
		if err != nil {
			return arg, err
		}
		if str == arg {
			// Done
			return str, nil
		}
		arg = str
	}
}

func ResolveAll(vars map[string]string, args []string) ([]string, error) {
	argArr := make([]string, len(args))
	for i, arg := range args {
		v, err := Resolve(vars, arg)
		if err != nil {
			return nil, err
		}
		argArr[i] = v
	}
	return argArr, nil
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
func HasPrimaryPeerJoinedChannel(channelID string, client *resmgmt.Client, orgUser mspApi.Identity, peer fabApi.Peer) (bool, error) {
	foundChannel := false
	response, err := client.QueryChannels(
		resmgmt.WithTargets(peer),
		resmgmt.WithRetry(retry.DefaultResMgmtOpts),
	)
	if err != nil {
		return false, fmt.Errorf("Error querying channel for primary peer: %s", err)
	}
	for _, responseChannel := range response.Channels {
		if responseChannel.ChannelId == channelID {
			foundChannel = true
			break
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
func IsChaincodeInstalled(client *resmgmt.Client, peer fabApi.Peer, name string) (bool, error) {
	chaincodeQueryResponse, err := client.QueryInstalledChaincodes(
		resmgmt.WithTargets(peer),
		resmgmt.WithRetry(retry.DefaultResMgmtOpts),
	)
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

func peersAsString(peers []fabApi.Peer) string {
	str := ""
	for i, p := range peers {
		str += p.URL()
		if i < len(peers)-1 {
			str += ", "
		}
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

func doResolve(vars map[string]string, arg string) (string, error) {
	if len(arg) <= 3 {
		return arg, nil
	}

	open := strings.Index(arg, "${")
	if open == -1 {
		return arg, nil
	}

	close := strings.Index(arg, "}")
	if close == -1 {
		return arg, errors.Errorf("expecting } for arg '%s'", arg)
	}

	// Check for array
	varName := arg[open+2 : close]
	ob := strings.Index(varName, "[")
	if ob == -1 {
		// Not an array
		return replace(arg, vars[varName], open, close), nil
	}

	cb := strings.Index(varName, "]")
	if cb == -1 {
		return arg, errors.Errorf("invalid arg '%s'", arg)
	}

	arrVar := varName[0:ob]
	values := vars[arrVar]

	if values == "" {
		return replace(arg, "", open, close), nil
	}

	index := varName[ob+1 : cb]

	vals := strings.Split(values, ",")
	i, err := strconv.Atoi(index)
	if err != nil {
		return arg, errors.Errorf("invalid index [%s] for arg '%s'", index, arg)
	}

	if i >= len(vals) {
		return arg, errors.Errorf("index [%d] out of range for arg '%s'", i, arg)
	}

	return replace(arg, vals[i], open, close), nil
}

func replace(arg, value string, open, close int) string {
	return arg[0:open] + value + arg[close+1:]
}
