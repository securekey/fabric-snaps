/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statemgr

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/txsnapservice/dbprovider"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("statemgr")

//GetState gets state from state DB using given channelID, namespace and key
func GetState(channelID, ccNamespace, key string) ([]byte, errors.Error) {

	db, err := dbprovider.GetStateDB(channelID)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Failed to get State DB")
	}

	err = db.Open()
	if err != nil {
		return nil, errors.WithMessage(errors.SystemError, err, "Failed to open db provider")
	}
	defer db.Close()
	defer logger.Debug("DB handle closed")

	logger.Debug("DB handle opened")

	vv, err := db.GetState(ccNamespace, key)
	if err != nil {
		return nil, errors.WithMessage(errors.SystemError, err, "Failed to get state from db")
	}

	if vv == nil {
		logger.Debugf("Query returned nil for namespace %s and key %s", ccNamespace, key)
		return nil, nil
	}

	logger.Debugf("Query returned %+v for namespace %s and key %s", vv.Value, ccNamespace, key)

	return vv.Value, nil
}
