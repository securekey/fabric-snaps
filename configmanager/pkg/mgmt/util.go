/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mgmt

import (
	"fmt"
	"strings"

	"github.com/securekey/fabric-snaps/configmanager/api"
)

const (
	keyDivider = "!"
)

//CreateConfigKey creates key using mspID, peerID and appName
func CreateConfigKey(mspID string, peerID string, appName string) (api.ConfigKey, error) {
	configKey := api.ConfigKey{MspID: mspID, PeerID: peerID, AppName: appName}
	if err := ValidateConfigKey(configKey); err != nil {
		return configKey, err
	}
	return configKey, nil
}

//ValidateConfigKey validates component parts of ConfigKey
func ValidateConfigKey(configKey api.ConfigKey) error {
	if len(configKey.MspID) == 0 {
		return fmt.Errorf("Cannot create config key using empty MspId")
	}
	if len(configKey.PeerID) == 0 {
		return fmt.Errorf("Cannot create config key using empty PeerID")
	}
	if len(configKey.AppName) == 0 {
		return fmt.Errorf("Cannot create config key using empty AppName")
	}
	return nil
}

//ConfigKeyToString converts configKey to string
func ConfigKeyToString(configKey api.ConfigKey) (string, error) {
	if err := ValidateConfigKey(configKey); err != nil {
		return "", err
	}
	return strings.Join([]string{configKey.MspID, configKey.PeerID, configKey.AppName}, keyDivider), nil
}

//StringToConfigKey converts string to ConfigKey{}
func StringToConfigKey(key string) (api.ConfigKey, error) {
	ck := api.ConfigKey{}
	keyParts := strings.Split(key, keyDivider)
	if len(keyParts) < 3 {
		return ck, fmt.Errorf("Invalid config key %v", key)
	}
	ck.MspID = keyParts[0]
	ck.PeerID = keyParts[1]
	ck.AppName = keyParts[2]
	return ck, nil
}
