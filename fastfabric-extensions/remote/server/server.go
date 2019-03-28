package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger"
	"github.com/hyperledger/fabric/fastfabric-extensions/cached"
	"github.com/hyperledger/fabric/fastfabric-extensions/remote"
	"github.com/hyperledger/fabric/protos/common"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
)

func main() {
	var server = flag.String("server", "localhost", "server address")
	var port = flag.String("port", "10000", "port")

	flag.Parse()

	fmt.Println("Starting server at", *server, ":", *port)

	startServer(fmt.Sprint(*server, ":", *port))
}

func startServer(address string) {

	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	remote.RegisterPersistentPeerServer(s, &server{})
	s.Serve(lis)
}

type server struct {
}


var peerledger ledger.PeerLedger

func (s *server) Store(ctx context.Context, block *common.Block) (*remote.Result, error) {
	if err := peerledger.CommitWithPvtData(&ledger.BlockAndPvtData{Block: cached.GetBlock(block)}); err != nil{
		return nil, err
	}

	return &remote.Result{}, nil
}


var input chan *common.Block

func (s *server) CreateLedger(ctx context.Context, genesisBlock *common.Block) (*remote.Result, error) {
	os.RemoveAll("/var/hyperledger/ledgersData/")
	//os.MkdirAll("/var/hyperledger/ledgersData/", 0700)
	fmt.Println("Constructing ledger")
	ledgerProvider, err := kvledger.NewProvider()
	if err != nil {
		panic(fmt.Errorf("Error in instantiating ledger provider: %s", err))
	}
	ledgerProvider.Initialize(nil)

	peerledger, err = ledgerProvider.Create(genesisBlock)
	if err != nil {
		return nil, err
	}

	fmt.Println("Ledger constructed")
	return &remote.Result{}, nil
}

