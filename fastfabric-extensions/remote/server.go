package remote

import (
	"context"
	"fmt"
	ledger2 "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/fastfabric-extensions/cached"
	"github.com/hyperledger/fabric/protos/common"
	"google.golang.org/grpc"
	"log"
	"net"
)

var storageServer = &server{peerledger:make(map[string]ledger.PeerLedger)}

func StartServer(address string) {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	RegisterStoragePeerServer(s, storageServer)
	go s.Serve(lis)
}

type server struct {
	ledgerProvider ledger.PeerLedgerProvider
	peerledger map[string]ledger.PeerLedger
	iterators []ledger2.ResultsIterator
}

func (s *server) IteratorNext(ctx context.Context, itr  *Iterator) (*common.Block, error) {
	res, err := s.iterators[itr.IteratorId].Next()
	return res.(*common.Block), err
}

func (s *server) IteratorClose(ctx context.Context, itr *Iterator) (*Result, error) {
	s.iterators[itr.IteratorId].Close()
	s.iterators[itr.IteratorId] = nil
	return &Result{}, nil
}

func (s *server) RetrieveBlocks(ctx context.Context, req *RetrieveBlocksRequest) (*Iterator, error) {
	itr, err := s.peerledger[req.LedgerId].GetBlockstore().RetrieveBlocks(req.StartNum)
	s.iterators = append(s.iterators, itr)
	return &Iterator{IteratorId:int32(len(s.iterators) -1)}, err
}

func (s *server) RetrieveTxValidationCodeByTxID(ctx context.Context, req *RetrieveTxValidationCodeByTxIDRequest) (*ValidationCode, error) {
	code, err := s.peerledger[req.LedgerId].GetBlockstore().RetrieveTxValidationCodeByTxID(req.TxID)
	return &ValidationCode{ValidationCode:int32(code)}, err
}

func (s *server) RetrieveBlockByTxID(ctx context.Context, req *RetrieveBlockByTxIDRequest) (*common.Block, error) {
	return s.peerledger[req.LedgerId].GetBlockstore().RetrieveBlockByTxID(req.TxID)
}

func (s *server) RetrieveTxByBlockNumTranNum(ctx context.Context, req *RetrieveTxByBlockNumTranNumRequest) (*common.Envelope, error) {
	return s.peerledger[req.LedgerId].GetBlockstore().RetrieveTxByBlockNumTranNum(req.BlockNo, req.TxNo)
}

func (s *server) RetrieveTxByID(ctx context.Context, req *RetrieveTxByIDRequest) (*common.Envelope, error) {
	return s.peerledger[req.LedgerId].GetBlockstore().RetrieveTxByID(req.TxID)
}

func (s *server) RetrieveBlockByNumber(ctx context.Context, req *RetrieveBlockByNumberRequest) (*common.Block, error) {
	return s.peerledger[req.LedgerId].GetBlockstore().RetrieveBlockByNumber(req.BlockNo)
}

func (s *server) GetBlockchainInfo(ctx context.Context, req *GetBlockchainInfoRequest) (*common.BlockchainInfo, error) {
	return s.peerledger[req.LedgerId].GetBlockstore().GetBlockchainInfo()
}

func (s *server) RetrieveBlockByHash(ctx context.Context, req *RetrieveBlockByHashRequest) (*common.Block, error) {
	return s.peerledger[req.LedgerId].GetBlockstore().RetrieveBlockByHash(req.BlockHash)
}

func (s *server) Store(ctx context.Context, req *StorageRequest) (*Result, error) {
	l := s.peerledger[req.LedgerId]
	if err := l.CommitWithPvtData(&ledger.BlockAndPvtData{Block: cached.GetBlock(req.Block)}); err != nil{
		return nil, err
	}

	return &Result{}, nil
}

func (s *server) CreateLedger(ctx context.Context, req *StorageRequest) (*Result, error) {
	fmt.Println("Constructing ledger")
	var err error
	if s.ledgerProvider == nil {
		return nil, fmt.Errorf("ledgerProvider must be set on this peer, is currently nil")
	}

	s.peerledger[req.LedgerId], err = s.ledgerProvider.Create(req.Block)
	if err != nil {
		return nil, err
	}

	fmt.Println("Ledger constructed")
	return &Result{}, nil
}

func SetLedgerProvider(provider ledger.PeerLedgerProvider){
	storageServer.ledgerProvider = provider
}

