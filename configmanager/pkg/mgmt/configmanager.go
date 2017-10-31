/*
   Copyright SecureKey Technologies Inc.
   This file contains software code that is the intellectual property of SecureKey.
   SecureKey reserves all rights in the code and you may not use it without written permission from SecureKey.
*/

package mgmt

//ConfigManager ....
type ConfigManager interface {
	SaveConfiguration(key string, configuration string) (saved bool, err error)
	GetConfiguration(key string) (config string, err error)
	DeleteConfiguration(key string) (deleted bool, err error)
}
