/*
   Copyright SecureKey Technologies Inc.
   This file contains software code that is the intellectual property of SecureKey.
   SecureKey reserves all rights in the code and you may not use it without written permission from SecureKey.
*/

package client

// ConfigClient is used to publish messages
type ConfigClient interface {
	Get(key string)
}
