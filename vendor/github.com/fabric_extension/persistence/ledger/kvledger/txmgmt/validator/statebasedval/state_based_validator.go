/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package statebasedval

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/validator/valinternal"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/protos/peer"
)

var logger = flogging.MustGetLogger("statebasedval")

// Validator validates a tx against the latest committed state
// and preceding valid transactions with in the same block
type Validator struct {
	db privacyenabledstate.DB
}

// NewValidator constructs StateValidator
func NewValidator(db privacyenabledstate.DB) *Validator {
	return &Validator{db}
}

// ValidateAndPrepareBatch implements method in Validator interface
func (v *Validator) ValidateAndPrepareBatch(block *valinternal.Block, doMVCCValidation bool) (*valinternal.PubAndHashUpdates, error) {
	updates := valinternal.NewPubAndHashUpdates()
	for _, tx := range block.Txs {

		if tx.ValidationCode == peer.TxValidationCode_VALID {
			logger.Debugf("Block [%d] Transaction index [%d] TxId [%s] marked as valid by state validator", block.Num, tx.IndexInBlock, tx.ID)
			committingTxHeight := version.NewHeight(block.Num, uint64(tx.IndexInBlock))
			updates.ApplyWriteSet(tx.RWSet, committingTxHeight)
		} else {
			logger.Warningf("Block [%d] Transaction index [%d] TxId [%s] marked as invalid by state validator. Reason code [%s]",
				block.Num, tx.IndexInBlock, tx.ID, tx.ValidationCode.String())
		}
	}
	return updates, nil
}

