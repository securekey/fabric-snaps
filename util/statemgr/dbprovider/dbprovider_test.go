/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbprovider

import (
	"fmt"
	"testing"

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestGetStateDB(t *testing.T) {
	setupMockDBProvider(nil)
	viper.Set("ledger.state.stateDatabase", "CouchDB")
	_, err := GetStateDB("test")
	assert.Nil(t, err, "Did not expect error")

	testErr := fmt.Errorf("test")
	setupMockDBProvider(testErr)
	_, err = GetStateDB("test")
	assert.NotNil(t, err)
	assert.Equal(t, testErr.Error(), err.Error())
}

func setupMockDBProvider(err error) {
	once.Do(func() {
	})
	vdbProvider = &mockDBProvider{err: err}
}

type mockDBProvider struct {
	err error
}

// GetDBHandle returns a handle to a PvtVersionedDB
func (m *mockDBProvider) GetDBHandle(id string) (statedb.VersionedDB, error) {
	return nil, m.err
}

// Close closes all the PvtVersionedDB instances and releases any resources held by VersionedDBProvider
func (m *mockDBProvider) Close() {

}
