/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bookkeeping

import (
	"fmt"

	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"

	ffstatedb "github.com/hyperledger/fabric/fastfabric-extensions/statedb"
)

// Category is an enum type for representing the bookkeeping of different type
type Category int

const (
	// PvtdataExpiry repersents the bookkeeping related to expiry of pvtdata because of BTL policy
	PvtdataExpiry Category = iota
	// MetadataPresenceIndicator maintains the bookkeeping about whether metadata is ever set for a namespace
	MetadataPresenceIndicator
)

// Provider provides handle to different bookkeepers for the given ledger
type Provider interface {
	// GetDBHandle returns a db handle that can be used for maintaining the bookkeeping of a given category
	GetDBHandle(ledgerID string, cat Category) *ffstatedb.DBHandle
	// Close closes the BookkeeperProvider
	Close()
}

type provider struct {
	dbProvider *ffstatedb.Provider
}

// NewProvider instantiates a new provider
func NewProvider() Provider {
	dbProvider := ffstatedb.NewProvider()
	return &provider{dbProvider: dbProvider}
}

// GetDBHandle implements the function in the interface 'BookkeeperProvider'
func (provider *provider) GetDBHandle(ledgerID string, cat Category) *ffstatedb.DBHandle {
	return provider.dbProvider.GetDBHandle(fmt.Sprintf(ledgerID+"/%d", cat))
}

// Close implements the function in the interface 'BookKeeperProvider'
func (provider *provider) Close() {
	provider.dbProvider.Close()
}

func getInternalBookkeeperPath() string {
	return ledgerconfig.GetInternalBookkeeperPath()
}
