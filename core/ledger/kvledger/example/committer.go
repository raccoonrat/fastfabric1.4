/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package example

import (
	"github.com/hyperledger/fabric/core/ledger"
)

// Committer a toy committer
type Committer struct {
	ledger ledger.PeerLedger
}

// ConstructCommitter constructs a committer for the example
func ConstructCommitter(ledger ledger.PeerLedger) *Committer {
	return &Committer{ledger}
}

// Commit commits the block
func (c *Committer) Commit(blockNo uint64) error {
	err := c.ledger.CommitWithPvtDataByNo(blockNo)
	if err != nil {
		return err
	}
	return nil
}
