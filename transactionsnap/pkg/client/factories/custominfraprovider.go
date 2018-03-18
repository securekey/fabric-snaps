/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package factories

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/provider/fabpvdr"
)

// CustomInfraProvider is will provide custom fabric primitives
type CustomInfraProvider struct {
	*fabpvdr.InfraProvider
}

// CreateEventService will return nil because the txnsnap will use local event service
func (f *CustomInfraProvider) CreateEventService(ctx fab.ClientContext, channelID string) (fab.EventService, error) {
	return nil, nil
}
