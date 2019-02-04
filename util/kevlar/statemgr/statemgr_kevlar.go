// +build kevlar

/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statemgr

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric/core/ledger/ledgermgmt"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("statemgr")

//GetState gets state from ledger using given channelID, namespace and key
func GetState(channelID, ccNamespace, key string) ([]byte, errors.Error) {

	ledger, err := ledgermgmt.OpenLedger(channelID)
	if err != nil {
		return nil, errors.WithMessage(errors.SystemError, err, "Failed to open ledger")
	}
	value, err := ledger.GetState(ccNamespace, key)
	if err != nil {
		return nil, errors.WithMessage(errors.SystemError, err, "Failed to get state")
	}

	logger.Debugf("Query returned %+v for namespace %s and key %s", value, ccNamespace, key)

	return value, nil
}
