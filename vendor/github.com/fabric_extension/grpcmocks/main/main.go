package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/fabric_extension/experiment"
	"github.com/fabric_extension/grpcmocks"
	"github.com/fabric_extension/persistence/ledger/kvledger"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/example"
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
	grpcmocks.RegisterCryptoServer(s, &server{})
	grpcmocks.RegisterStorageServer(s, &server{})
	s.Serve(lis)
}

type server struct {
}

func (s *server) QueryBalances(ctx context.Context, acc *grpcmocks.Accounts) (*grpcmocks.Balances, error) {
	bal, err := app.QueryBalances(acc.Accs)
	balances := make([]int32, len(bal))
	mess := &grpcmocks.Balances{}
	if err == nil {
		for i, b := range bal {
			balances[i] = int32(b)
		}
		mess.Bal = balances
	}
	return mess, err
}

func (s *server) Init(ctx context.Context, init *grpcmocks.EndorseInit) (*common.Envelope, error) {
	i := make(map[string]int, len(init.InitBalances))
	for account, balance := range init.InitBalances {
		i[account] = int(balance)
	}
	return app.Init(i)
}

var endorsingcount = 0

func (s *server) Endorse(ctx context.Context, tr *grpcmocks.Transfer) (*common.Envelope, error) {
	experiment.IsEndorser = true
	if endorsingcount < 10 {
		endorsingcount++
	}
	return app.TransferFunds(tr.From, tr.To, int(tr.Amount))
}

var peerledger ledger.PeerLedger

func (s *server) Store(ctx context.Context, block *common.Block) (*grpcmocks.Result, error) {
	input <- block
	return &grpcmocks.Result{}, nil
}

var app *example.App

var input chan *common.Block

func (s *server) CreateLedger(ctx context.Context, genesisBlock *common.Block) (*grpcmocks.Result, error) {
	os.RemoveAll("/var/hyperledger/ledgersData/")
	//os.MkdirAll("/var/hyperledger/ledgersData/", 0700)
	fmt.Println("Constructing ledger")
	ledgerProvider, err := kvledger.NewProvider()
	if err != nil {
		panic(fmt.Errorf("Error in instantiating ledger provider: %s", err))
	}
	ledgerProvider.Initialize([]ledger.StateListener{})

	peerledger, err = ledgerProvider.Create(genesisBlock)
	if err != nil {
		return nil, err
	}

	app = example.ConstructAppInstance(peerledger)

	input = make(chan *common.Block, 100000)
	go waitForCommit()
	fmt.Println("Ledger constructed")
	return &grpcmocks.Result{}, nil
}

func waitForCommit() {
	for {
		peerledger.CommitWithPvtData(&ledger.BlockAndPvtData{Block: <-input})
	}
}

func (s *server) CompareHash(ctx context.Context, m *grpcmocks.CompareMessage) (*grpcmocks.Result, error) {
	return &grpcmocks.Result{}, nil
}

func (s *server) Verify(ctx context.Context, t *grpcmocks.Transaction) (*grpcmocks.Result, error) {
	return &grpcmocks.Result{}, nil
}
