/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package configkeyutil

import (
	"encoding/json"

	"github.com/hyperledger/fabric-sdk-go/pkg/errors"
	mgmtapi "github.com/securekey/fabric-snaps/configmanager/api"
)

// Unmarshal unmarshals a ConfigKey from bytes
func Unmarshal(configKeyBytes []byte) (*mgmtapi.ConfigKey, error) {
	if len(configKeyBytes) == 0 {
		return nil, errors.New("config is empty")
	}
	configKey := &mgmtapi.ConfigKey{}
	if err := json.Unmarshal(configKeyBytes, &configKey); err != nil {
		return nil, errors.New("invalid config key specified")
	}
	return configKey, nil
}
