package fsblkstorage

import (
	"context"
	"github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/fastfabric-extensions/remote"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
)

func newFsBlockStore(ledgerId string) *BlockStoreImpl {
	return &BlockStoreImpl{ledgerId : ledgerId, client: remote.GetStoragePeerClient()}
}



type BlockStoreImpl struct {
	client   remote.StoragePeerClient
	ledgerId string
}

func (BlockStoreImpl) AddBlock(block *common.Block) error {
	return nil
}

func (b BlockStoreImpl) GetBlockchainInfo() (*common.BlockchainInfo, error) {
	return b.client.GetBlockchainInfo(context.Background(), &remote.GetBlockchainInfoRequest{LedgerId:b.ledgerId})
}

type Iterator struct {
	itr   *remote.Iterator
	client remote.StoragePeerClient
}

func (i Iterator) Next() (ledger.QueryResult, error) {
	return i.client.IteratorNext(context.Background(), i.itr)
}

func (i Iterator) Close() {
	i.client.IteratorClose(context.Background(), i.itr)
}

func (b BlockStoreImpl) RetrieveBlocks(startNum uint64) (ledger.ResultsIterator, error) {
	itr, err :=  b.client.RetrieveBlocks(context.Background(), &remote.RetrieveBlocksRequest{
		LedgerId:b.ledgerId,
		StartNum:startNum})

	return &Iterator{itr: itr, client:b.client}, err
}

func (b BlockStoreImpl) RetrieveBlockByHash(blockHash []byte) (*common.Block, error) {
	return b.client.RetrieveBlockByHash(context.Background(), &remote.RetrieveBlockByHashRequest{
		LedgerId:b.ledgerId,
		BlockHash:blockHash})
}

func (b BlockStoreImpl) RetrieveBlockByNumber(blockNum uint64) (*common.Block, error) {
	return b.client.RetrieveBlockByNumber(context.Background(), &remote.RetrieveBlockByNumberRequest{
		LedgerId:b.ledgerId,
		BlockNo:blockNum})
}

func (b BlockStoreImpl) RetrieveTxByID(txID string) (*common.Envelope, error) {
	return b.client.RetrieveTxByID(context.Background(), &remote.RetrieveTxByIDRequest{
		LedgerId:b.ledgerId,
		TxID:txID})
}

func (b BlockStoreImpl) RetrieveTxByBlockNumTranNum(blockNum uint64, tranNum uint64) (*common.Envelope, error) {
	return b.client.RetrieveTxByBlockNumTranNum(context.Background(), &remote.RetrieveTxByBlockNumTranNumRequest{
		LedgerId:b.ledgerId,
		BlockNo:blockNum,
		TxNo:tranNum})
}

func (b BlockStoreImpl) RetrieveBlockByTxID(txID string) (*common.Block, error) {
	return b.client.RetrieveBlockByTxID(context.Background(), &remote.RetrieveBlockByTxIDRequest{
		LedgerId: b.ledgerId,
		TxID: txID})
}

func (b BlockStoreImpl) RetrieveTxValidationCodeByTxID(txID string) (peer.TxValidationCode, error) {
	code, err := b.client.RetrieveTxValidationCodeByTxID(context.Background(), &remote.RetrieveTxValidationCodeByTxIDRequest{
		LedgerId:b.ledgerId,
		TxID:txID})
	return peer.TxValidationCode(code.ValidationCode), err
}

func (b BlockStoreImpl) Shutdown() {
}

