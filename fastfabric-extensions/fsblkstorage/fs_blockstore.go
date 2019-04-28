package fsblkstorage

import (
	"context"
	"github.com/hyperledger/fabric/common/ledger"
	coreLedger "github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/fastfabric-extensions/cached"
	"github.com/hyperledger/fabric/fastfabric-extensions/remote"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"sync"
)

func newFsBlockStore(ledgerId string) *BlockStoreImpl {
	return &BlockStoreImpl{ledgerId : ledgerId, client: remote.GetStoragePeerClient(), txCache: make(map[string]bool)}
}

type BlockStoreImpl struct {
	client   remote.StoragePeerClient
	ledgerId string
	txCache map[string]bool
	items map[string][]byte
	lock  sync.RWMutex
}

func (b BlockStoreImpl) AddBlock(block *cached.Block) error {
	envs, _ := block.UnmarshalAllEnvelopes()
	b.lock.Lock()
	defer b.lock.Unlock()
	for _,env := range envs{
		pl, _ := env.UnmarshalPayload()
		chdr, _ := pl.Header.UnmarshalChannelHeader()
		b.txCache[chdr.TxId] = true
	}
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

func (b BlockStoreImpl) checkCache(txID string) bool{
	b.lock.RLock()
	defer b.lock.RUnlock()
	 _, ok := b.txCache[txID]
	 return ok
}
func (b BlockStoreImpl) RetrieveTxByID(txID string) (*common.Envelope, error) {
	if ok := b.checkCache(txID);!ok{
		return nil, coreLedger.NotFoundInIndexErr("")
	}
	tx, err := b.client.RetrieveTxByID(context.Background(), &remote.RetrieveTxByIDRequest{
		LedgerId:b.ledgerId,
		TxID:txID})
	if err.Error() == "rpc error: code = Unknown desc = Entry not found in index" {
		return tx, coreLedger.NotFoundInIndexErr("")
	}
	return tx, err
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

