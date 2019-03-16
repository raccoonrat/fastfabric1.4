/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package valimpl

import (
	"github.com/fabric_extension/block_cache"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/txmgr"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/validator"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/validator/statebasedval"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/validator/valinternal"
)

var logger = flogging.MustGetLogger("valimpl")

// DefaultImpl implements the interface validator.Validator
// This performs the common tasks that are independent of a particular scheme of validation
// and for actual validation of the public rwset, it encloses an internal validator (that implements interface
// valinternal.InternalValidator) such as statebased validator
type DefaultImpl struct {
	txmgr txmgr.TxMgr
	db    privacyenabledstate.DB
	valinternal.FastInternalValidator
	valinternal.InternalValidator
}

// NewStatebasedValidator constructs a validator that internally manages statebased validator and in addition
// handles the tasks that are agnostic to a particular validation scheme such as parsing the block and handling the pvt data
func NewStatebasedValidator(txmgr txmgr.TxMgr, db privacyenabledstate.DB) validator.FastValidator {
	val := statebasedval.NewValidator(db)
	return &DefaultImpl{txmgr, db, val, val}
}

// ValidateAndPrepareBatch implements the function in interface validator.Validator
func (impl *DefaultImpl) ValidateAndPrepareBatch(blockAndPvtdata *ledger.BlockAndPvtData,
	doMVCCValidation bool) (*privacyenabledstate.UpdateBatch, error) {
	rawblock := blockAndPvtdata.Block
	logger.Debugf("ValidateAndPrepareBatch() for block number = [%d]", rawblock.Header.Number)
	block,_:=blocks.Cache.Get(rawblock.Header.Number)

	var internalBlock *valinternal.Block
	var pubAndHashUpdates *valinternal.PubAndHashUpdates
	var pvtUpdates *privacyenabledstate.PvtUpdateBatch
	var err error

	logger.Debug("preprocessing ProtoBlock...")
	if internalBlock, err = preprocessProtoBlock(impl.txmgr, impl.db.ValidateKeyValue, block, doMVCCValidation); err != nil {
		return nil, err
	}

	if pubAndHashUpdates, err = impl.InternalValidator.ValidateAndPrepareBatch(internalBlock, doMVCCValidation); err != nil {
		return nil, err
	}
	logger.Debug("validating rwset...")
	if pvtUpdates, err = validateAndPreparePvtBatch(internalBlock, blockAndPvtdata.BlockPvtData); err != nil {
		return nil, err
	}
	logger.Debug("postprocessing ProtoBlock...")
	postprocessProtoBlock(rawblock, internalBlock)
	logger.Debug("ValidateAndPrepareBatch() complete")
	return &privacyenabledstate.UpdateBatch{
		PubUpdates:  pubAndHashUpdates.PubUpdates,
		HashUpdates: pubAndHashUpdates.HashUpdates,
		PvtUpdates:  pvtUpdates,
	}, nil
}


func (impl *DefaultImpl) ValidateAndPrepareBatchByNo(blockNo uint64,
	doMVCCValidation bool) (*privacyenabledstate.UpdateBatch, error) {
	logger.Debugf("ValidateAndPrepareBatch() for block number = [%d]", blockNo)

	var pubAndHashUpdates *valinternal.PubAndHashUpdates
	var pvtUpdates *privacyenabledstate.PvtUpdateBatch
	var err error

	logger.Debug("preprocessing ProtoBlock...")
	if err = preprocessProtoBlockByNo(impl.txmgr, impl.db.ValidateKeyValue, blockNo, doMVCCValidation); err != nil {
		return nil, err
	}

	if pubAndHashUpdates, err = impl.FastInternalValidator.ValidateAndPrepareBatchByNo(blockNo, doMVCCValidation); err != nil {
		return nil, err
	}
	logger.Debug("validating rwset...")
	if pvtUpdates, err = validateAndPreparePvtBatchByNo( blockNo); err != nil {
		return nil, err
	}
	logger.Debug("ValidateAndPrepareBatch() complete")
	return &privacyenabledstate.UpdateBatch{
		PubUpdates:  pubAndHashUpdates.PubUpdates,
		HashUpdates: pubAndHashUpdates.HashUpdates,
		PvtUpdates:  pvtUpdates,
	}, nil
}