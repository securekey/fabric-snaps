/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package configdata

/*
	Variables passed in with ldflags
   	Example :  -X github.com/securekey/fabric-ext/configurationsnap/cmd/configurationscc/configdata.PublicKeyForLogging=SAMPLE_KEY
*/

//PublicKeyForLogging used as a public key for private logging
var PublicKeyForLogging string

//KeyIDForLogging key id matching public key for private logging
var KeyIDForLogging string

//EncryptLogging encrypt logging flag
var EncryptLogging string
