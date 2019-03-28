package remote

import (
	"github.com/hyperledger/fabric/fastfabric-extensions/remote"
	"google.golang.org/grpc"
)

var conn grpc.ClientConn

var client remote.PersistentPeerClient

func StartPersistentPeerClient(address string) error {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return err
	}
	client = remote.NewPersistentPeerClient(conn)
	return nil
}

func GetPersistentPeerClient() remote.PersistentPeerClient{
	return client
}
