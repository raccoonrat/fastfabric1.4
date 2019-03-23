package fsblkstorage

import (
	"github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/fastfabric-extensions/statedb"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
)

func newFsBlockStore(id string, indexConfig *blkstorage.IndexConfig,
	dbHandle *statedb.DBHandle) *BlockStore{
	return &BlockStore{}
}

type BlockStore struct{}

func (BlockStore) AddBlock(block *common.Block) error {
	panic("implement me")
}

func (BlockStore) GetBlockchainInfo() (*common.BlockchainInfo, error) {
	panic("implement me")
}

func (BlockStore) RetrieveBlocks(startNum uint64) (ledger.ResultsIterator, error) {
	panic("implement me")
}

func (BlockStore) RetrieveBlockByHash(blockHash []byte) (*common.Block, error) {
	panic("implement me")
}

func (BlockStore) RetrieveBlockByNumber(blockNum uint64) (*common.Block, error) {
	panic("implement me")
}

func (BlockStore) RetrieveTxByID(txID string) (*common.Envelope, error) {
	panic("implement me")
}

func (BlockStore) RetrieveTxByBlockNumTranNum(blockNum uint64, tranNum uint64) (*common.Envelope, error) {
	panic("implement me")
}

func (BlockStore) RetrieveBlockByTxID(txID string) (*common.Block, error) {
	panic("implement me")
}

func (BlockStore) RetrieveTxValidationCodeByTxID(txID string) (peer.TxValidationCode, error) {
	panic("implement me")
}

func (BlockStore) Shutdown() {
	panic("implement me")
}

