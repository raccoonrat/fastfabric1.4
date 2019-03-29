package remote

import (
	"context"
	"fmt"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/fastfabric-extensions/cached"
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
}

func (s *server) Store(ctx context.Context, req *StorageRequest) (*Result, error) {
	l := s.peerledger[req.ChannelId]
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

	s.peerledger[req.ChannelId], err = s.ledgerProvider.Create(req.Block)
	if err != nil {
		return nil, err
	}

	fmt.Println("Ledger constructed")
	return &Result{}, nil
}

func SetLedgerProvider(provider ledger.PeerLedgerProvider){
	storageServer.ledgerProvider = provider
}

