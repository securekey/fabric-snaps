/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package factories

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/context/api/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/provider/fabpvdr"
)

// CustomFabricProvider is will provide custom fabric provider
type CustomFabricProvider struct {
	*fabpvdr.FabricProvider
}

// CreateEventHub will return nil because the txnsnap will use local event service
func (f *CustomFabricProvider) CreateEventHub(ic fab.IdentityContext, channelID string) (fab.EventHub, error) {
	return nil, nil
}
