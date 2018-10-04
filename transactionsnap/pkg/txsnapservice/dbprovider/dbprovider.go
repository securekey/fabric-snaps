/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbprovider

import (
	"sync"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("txnsnap")

var vdbProvider statedb.VersionedDBProvider
var dbProviderErr error
var once sync.Once

// GetStateDB returns a handle to the local state database. Only CouchDB is supported.
// connections to the database are cached.
// The handle may be closed and discarded after use.
func GetStateDB(channelID string) (privacyenabledstate.DB, error) {
	if !ledgerconfig.IsCouchDBEnabled() {
		return nil, errors.Errorf(errors.SystemError, "Local query is only supported on CouchDB")
	}
	once.Do(func() {
		logger.Info("Creating StateDB provider")
		vdbProvider, dbProviderErr = statecouchdb.NewVersionedDBProvider()
		if dbProviderErr != nil {
			logger.Warnf("Error creating StateDB provider %s", dbProviderErr)
		}
	})

	var err error
	if vdbProvider == nil {
		vdbProvider, err = statecouchdb.NewVersionedDBProvider()
		if err != nil {
			return nil, err
		}
	}
	db, err := vdbProvider.GetDBHandle(channelID)
	if err != nil {
		return nil, err
	}
	commonStorageDb, err := privacyenabledstate.NewCommonStorageDB(db, "", nil)
	if err != nil {
		return nil, err
	}
	logger.Debugf("Got State DB handle for channel %s", channelID)

	return commonStorageDb, nil
}
