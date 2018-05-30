/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbprovider

import (
	"sync"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("txnsnap")

var stateDBProvider privacyenabledstate.DBProvider
var dbProviderErr error
var once sync.Once

// GetStateDB returns a handle to the local state database. Only CouchDB is supported.
// connections to the database are cached.
// The handle may be closed and discarded after use.
func GetStateDB(channelID string) (privacyenabledstate.DB, error) {
	dbProvider, err := getStateDBProviderInstance()
	if err != nil {
		return nil, err
	}

	db, err := dbProvider.GetDBHandle(channelID)
	if err != nil {
		return nil, err
	}

	logger.Debugf("Got State DB handle for channel %s", channelID)

	return db, nil
}

func getStateDBProviderInstance() (privacyenabledstate.DBProvider, error) {
	if !ledgerconfig.IsCouchDBEnabled() {
		return nil, errors.Errorf(errors.GeneralError, "Local query is only supported on CouchDB")
	}

	once.Do(func() {
		logger.Info("Creating StateDB provider")
		stateDBProvider, dbProviderErr = privacyenabledstate.NewCommonStorageDBProvider()
		if dbProviderErr != nil {
			logger.Warnf("Error creating StateDB provider %s", dbProviderErr)
		}
	})

	return stateDBProvider, dbProviderErr
}
