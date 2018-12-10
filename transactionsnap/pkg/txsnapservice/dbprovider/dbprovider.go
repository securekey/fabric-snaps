/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbprovider

import (
	"sync"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecachedstore"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statekeyindex"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("txnsnap")

var stateDBProvider statedb.VersionedDBProvider
var dbProviderErr error
var once sync.Once

// GetStateDB returns a handle to the local state database. Only CouchDB is supported.
// connections to the database are cached.
// The handle may be closed and discarded after use.
func GetStateDB(channelID string) (statedb.VersionedDB, error) {
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

func getStateDBProviderInstance() (statedb.VersionedDBProvider, error) {
	if !ledgerconfig.IsCouchDBEnabled() {
		return nil, errors.Errorf(errors.SystemError, "Local query is only supported on CouchDB")
	}

	once.Do(func() {
		logger.Info("Creating StateDB provider")
		vdbProvider, err := statecouchdb.NewVersionedDBProvider()
		if err != nil {
			logger.Warnf("Error creating StateDB provider %s", err)
		}
		stateKeyIndexProvider := statekeyindex.NewProvider()

		stateDBProvider = statecachedstore.NewProvider(
			vdbProvider,
			stateKeyIndexProvider)

	})

	return stateDBProvider, dbProviderErr
}
