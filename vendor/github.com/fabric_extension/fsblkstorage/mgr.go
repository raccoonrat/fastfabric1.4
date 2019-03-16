package fsblkstorage

import (
	"github.com/fabric_extension/hashtable"
	"github.com/hyperledger/fabric/common/ledger"
	l "github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
)

var txIdCache *hashtable.ValueHashtable

type MockBlockfileMgr struct {
}

// AddBlock adds a new block
func (mgr *MockBlockfileMgr) AddBlock(block *common.Block) error {
	return nil
}

func AddTxId(id string) {
	if txIdCache == nil{
		txIdCache = hashtable.New()
	}
	txIdCache.Put([]byte(id), nil)
}

// GetBlockchainInfo returns the current info about blockchain
func (mgr *MockBlockfileMgr) GetBlockchainInfo() (*common.BlockchainInfo, error) {
	return &common.BlockchainInfo{Height:0}, nil
}

// RetrieveBlocks returns an iterator that can be used for iterating over a range of blocks
func (mgr *MockBlockfileMgr) RetrieveBlocks(startNum uint64) (ledger.ResultsIterator, error) {
	return nil, nil
}

// RetrieveBlockByHash returns the block for given block-hash
func (mgr *MockBlockfileMgr) RetrieveBlockByHash(blockHash []byte) (*common.Block, error) {
	return nil, nil
}

// RetrieveBlockByNumber returns the block at a given blockchain height
func (mgr *MockBlockfileMgr) RetrieveBlockByNumber(blockNum uint64) (*common.Block, error) {
	return nil, nil
}

// RetrieveTxByID returns a transaction for given transaction id
func (mgr *MockBlockfileMgr) RetrieveTransactionByID(txID string) (*common.Envelope, error) {
	if txIdCache == nil{
		txIdCache = hashtable.New()
	}

	if _, err := txIdCache.Get([]byte(txID));err != nil {
		return nil,l.NotFoundInIndexErr("")
	}
	return nil, nil

}

// RetrieveTxByID returns a transaction for given transaction id
func (mgr *MockBlockfileMgr) RetrieveTransactionByBlockNumTranNum(blockNum uint64, tranNum uint64) (*common.Envelope, error) {
	return nil, nil
}

func (mgr *MockBlockfileMgr) RetrieveBlockByTxID(txID string) (*common.Block, error) {
	panic("RetrieveBlockByTxID")
	return nil, nil
}

func (mgr *MockBlockfileMgr) RetrieveTxValidationCodeByTxID(txID string) (peer.TxValidationCode, error) {
	panic("RetrieveTxValidationCodeByTxID")
	return peer.TxValidationCode_INVALID_OTHER_REASON, nil
}

// Shutdown shuts down the block store
func (mgr *MockBlockfileMgr) Close(){
}