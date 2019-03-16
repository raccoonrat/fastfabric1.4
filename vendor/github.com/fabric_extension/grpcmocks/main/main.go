package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/fabric_extension/grpcmocks"
	"github.com/hyperledger/fabric/protos/common"
	"google.golang.org/grpc"
	"log"
	"net"
)

func main() {
	var server = flag.String("server", "localhost", "server address")
	var port = flag.String("port", "10000", "port")

	flag.Parse()

	fmt.Println("Starting server at",*server,":", *port)

	startServer(fmt.Sprint(*server,":", *port))
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

func (s *server) Store(ctx context.Context, block *common.Block) (*grpcmocks.Result, error) {
	return &grpcmocks.Result{}, nil
}

func (s *server) CompareHash(ctx context.Context, m *grpcmocks.CompareMessage) (*grpcmocks.Result, error) {
	return &grpcmocks.Result{}, nil
}

func (s *server) Verify(ctx context.Context, t *grpcmocks.Transaction) (*grpcmocks.Result, error) {
	return &grpcmocks.Result{}, nil
}
