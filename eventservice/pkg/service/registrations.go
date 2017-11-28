/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"regexp"

	eventapi "github.com/securekey/fabric-snaps/eventservice/api"
)

type channelRegistration struct {
	respch chan<- *eventapi.RegistrationResponse
}

type blockRegistration struct {
	eventch chan<- *eventapi.BlockEvent
}

type filteredBlockRegistration struct {
	eventch chan<- *eventapi.FilteredBlockEvent
}

type ccRegistration struct {
	ccID        string
	eventFilter string
	eventRegExp *regexp.Regexp
	eventch     chan<- *eventapi.CCEvent
}

type txRegistration struct {
	txID    string
	eventch chan<- *eventapi.TxStatusEvent
}
