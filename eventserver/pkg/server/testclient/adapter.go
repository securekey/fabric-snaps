/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package testclient

import (
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/eventserver/api"
)

// ChannelAdapter is the interface by which a fabric channel service client
// registers for interested channels and receives events from the channel
// service server. This is a reference Go implementation of a channel service
// client
type ChannelAdapter interface {
	GetInterestedChannels() ([]string, error)
	GetInterestedEvents() ([]*peer.Interest, error)
	Recv(msg *api.ChannelServiceResponse) (bool, error)
	Disconnected(err error)
}
