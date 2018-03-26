/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package configcache

import (
	"strings"

	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazycache"
	"github.com/securekey/fabric-snaps/util/errors"
	"github.com/spf13/viper"
)

// Cache manages a cache of Vipers
type Cache struct {
	cache       *lazycache.Cache
	defaultPath string
}

// New returns a new config cache for the given path
func New(name, envPrefix, defaultPath string) *Cache {
	return &Cache{
		defaultPath: defaultPath,
		cache: lazycache.New("Peer_Config_Cache", func(key lazycache.Key) (interface{}, error) {
			return newConfig(key.String(), name, envPrefix)
		}),
	}
}

// Get returns the config for the given path.
func (c *Cache) Get(path string) (*viper.Viper, error) {
	if path == "" {
		path = c.defaultPath
	}
	config, err := c.cache.Get(lazycache.NewStringKey(path))
	if err != nil {
		return nil, err
	}
	return config.(*viper.Viper), nil
}

func newConfig(path, name, envPrefix string) (*viper.Viper, error) {
	replacer := strings.NewReplacer(".", "_")
	config := viper.New()
	config.AddConfigPath(path)
	config.SetConfigName(name)
	config.SetEnvPrefix(envPrefix)
	config.AutomaticEnv()
	config.SetEnvKeyReplacer(replacer)
	err := config.ReadInConfig()
	if err != nil {
		return nil, errors.Wrapf(errors.GeneralError, err, "Error reading config file [%s]", path)
	}
	return config, nil
}
