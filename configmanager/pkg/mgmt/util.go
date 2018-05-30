/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mgmt

import (
	"strings"

	"github.com/securekey/fabric-snaps/util/errors"

	"github.com/securekey/fabric-snaps/configmanager/api"
)

const (
	//KeyDivider is used to separate key parts
	KeyDivider = "!"
)

//CreateConfigKey creates key using mspID, peerID, appName and version
func CreateConfigKey(mspID, peerID, appName, version, componentName string) (api.ConfigKey, error) {
	configKey := api.ConfigKey{MspID: mspID, PeerID: peerID, AppName: appName, Version: version, ComponentName: componentName}
	if err := ValidateConfigKey(configKey); err != nil {
		return configKey, err
	}
	return configKey, nil
}

//ValidateConfigKey validates component parts of ConfigKey
func ValidateConfigKey(configKey api.ConfigKey) error {
	if len(configKey.MspID) == 0 {
		return errors.New(errors.GeneralError, "Cannot create config key using empty MspId")
	}
	if len(configKey.PeerID) == 0 && len(configKey.AppName) == 0 {
		return errors.New(errors.GeneralError, "Cannot create config key using empty PeerID aand an empty App list")
	}
	if len(configKey.PeerID) > 0 && len(configKey.AppName) == 0 {
		return errors.New(errors.GeneralError, "Cannot create config key using empty AppName")
	}
	if len(configKey.Version) == 0 {
		return errors.New(errors.GeneralError, "Cannot create config key using empty Version")
	}
	return nil
}

//ConfigKeyToString converts configKey to string
func ConfigKeyToString(configKey api.ConfigKey) (string, error) {
	if err := ValidateConfigKey(configKey); err != nil {
		return "", errors.WithMessage(errors.GeneralError, err, "Config Key is not valid")
	}
	return strings.Join([]string{configKey.MspID, configKey.PeerID, configKey.AppName, configKey.Version}, KeyDivider), nil
}

//StringToConfigKey converts string to ConfigKey{}
func StringToConfigKey(key string) (api.ConfigKey, error) {
	ck := api.ConfigKey{}
	keyParts := strings.Split(key, KeyDivider)
	if len(keyParts) < 3 {
		return ck, errors.Errorf(errors.GeneralError, "Invalid config key %v", key)
	}
	ck.MspID = keyParts[0]
	ck.PeerID = keyParts[1]
	ck.AppName = keyParts[2]
	ck.Version = keyParts[3]
	return ck, nil
}
