/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package factories

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/context/api/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/provider/fabpvdr"
)

// CustomInfraProvider is will provide custom fabric primitives
type CustomInfraProvider struct {
	*fabpvdr.InfraProvider
}

// CreateEventHub will return nil because the txnsnap will use local event service
func (f *CustomInfraProvider) CreateEventHub(ic fab.IdentityContext, channelID string) (fab.EventHub, error) {
	return nil, nil
}
