package remote

import (
	"context"
	"fmt"
	"github.com/hyperledger/fabric/common/flogging"
	ledger2 "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/fastfabric-extensions"
	"github.com/hyperledger/fabric/fastfabric-extensions/cached"
	"github.com/hyperledger/fabric/protos/common"
	"google.golang.org/grpc"
	"net"
)

var remoteLogger = flogging.MustGetLogger("remote")
var storageServer = &server{peerLedger: make(map[string]ledger.PeerLedger)}

func StartServer(address string) {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		remoteLogger.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	RegisterStoragePeerServer(s, storageServer)
	go s.Serve(lis)
	remoteLogger.Infof("start listening for blocks")
}

type server struct {
	peerLedger   map[string]ledger.PeerLedger
	iterators    []ledger2.ResultsIterator
	createLedger createFn
}

func (s *server) IteratorNext(ctx context.Context, itr  *Iterator) (*common.Block, error) {
	res, err := s.iterators[itr.IteratorId].Next()
	if res == nil {
		return nil, err
	}
	return res.(*common.Block), err
}

func (s *server) IteratorClose(ctx context.Context, itr *Iterator) (*Result, error) {
	s.iterators[itr.IteratorId].Close()
	s.iterators[itr.IteratorId] = nil
	return &Result{}, nil
}

func (s *server) RetrieveBlocks(ctx context.Context, req *RetrieveBlocksRequest) (*Iterator, error) {
	if store:= s.getBlockstore(req.LedgerId); store != nil {
		itr, err := store.RetrieveBlocks(req.StartNum)
		s.iterators = append(s.iterators, itr)
		return &Iterator{IteratorId: int32(len(s.iterators) - 1)}, err
	}
	return nil, fmt.Errorf("store not initialized yet")
}

func (s *server) RetrieveTxValidationCodeByTxID(ctx context.Context, req *RetrieveTxValidationCodeByTxIDRequest) (*ValidationCode, error) {
	if store:= s.getBlockstore(req.LedgerId); store != nil {
		code, err := store.RetrieveTxValidationCodeByTxID(req.TxID)
		return &ValidationCode{ValidationCode:int32(code)}, err
	}
	return nil, fmt.Errorf("store not initialized yet")
}

func (s *server) RetrieveBlockByTxID(ctx context.Context, req *RetrieveBlockByTxIDRequest) (*common.Block, error) {
	if store:= s.getBlockstore(req.LedgerId); store != nil {
		return store.RetrieveBlockByTxID(req.TxID)
	}
	return nil, fmt.Errorf("store not initialized yet")
}

func (s *server) getBlockstore(ledgerId string) fastfabric_extensions.BlockStore {
	if l, ok := s.peerLedger[ledgerId]; ok {
		return l.GetBlockstore()
	}
	return nil
}

func (s *server) RetrieveTxByBlockNumTranNum(ctx context.Context, req *RetrieveTxByBlockNumTranNumRequest) (*common.Envelope, error) {
	if store:= s.getBlockstore(req.LedgerId); store != nil {
		return store.RetrieveTxByBlockNumTranNum(req.BlockNo, req.TxNo)
	}
	return nil, fmt.Errorf("store not initialized yet.")
}

func (s *server) RetrieveTxByID(ctx context.Context, req *RetrieveTxByIDRequest) (*common.Envelope, error) {
	if store:= s.getBlockstore(req.LedgerId); store != nil {
		return store.RetrieveTxByID(req.TxID)
	}
	return nil, fmt.Errorf("store not initialized yet.")
}

func (s *server) RetrieveBlockByNumber(ctx context.Context, req *RetrieveBlockByNumberRequest) (*common.Block, error) {
	if store:= s.getBlockstore(req.LedgerId); store != nil {
		return store.RetrieveBlockByNumber(req.BlockNo)
	}
	return nil, fmt.Errorf("store not initialized yet")
}

func (s *server) GetBlockchainInfo(ctx context.Context, req *GetBlockchainInfoRequest) (*common.BlockchainInfo, error) {
	if store:= s.getBlockstore(req.LedgerId); store != nil {
		return store.GetBlockchainInfo()
	}
	return &common.BlockchainInfo{
		Height:            0,
		CurrentBlockHash:  nil,
		PreviousBlockHash: nil}, nil
}

func (s *server) RetrieveBlockByHash(ctx context.Context, req *RetrieveBlockByHashRequest) (*common.Block, error) {
	if store:= s.getBlockstore(req.LedgerId); store != nil {
		return store.RetrieveBlockByHash(req.BlockHash)
	}
	return nil, fmt.Errorf("store not initialized yet")
}

func (s *server) Store(ctx context.Context, req *StorageRequest) (*Result, error) {
	l := s.peerLedger[req.LedgerId]
	if l == nil {
		return nil, fmt.Errorf("store not initialized yet")
	}
	block := cached.GetBlock(req.Block)
	if err := l.CommitWithPvtData(&ledger.BlockAndPvtData{Block: block}); err != nil{
		return nil, err
	}

	return &Result{}, nil
}

func (s *server) CreateLedger(ctx context.Context, req *StorageRequest) (*Result, error) {
	var err error

	s.peerLedger[req.LedgerId], err = s.createLedger(req.Block)
	if err != nil {
		return nil, err
	}

	remoteLogger.Info("ledger constructed:", req.LedgerId)
	return &Result{}, nil
}

func SetCreateLedgerFunc(fn createFn){
	storageServer.createLedger = fn
}

type createFn func(genesisBlock *common.Block) (ledger.PeerLedger, error)

func SetLedger(ledgerId string, ledger ledger.PeerLedger){
	storageServer.peerLedger[ledgerId] = ledger
}