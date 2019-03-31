package remote

import (
	"google.golang.org/grpc"
)

var endorserClients = make([]StoragePeerClient,0)
var storageClient StoragePeerClient


func StartStoragePeerClient(address string) error {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return err
	}
	storageClient = NewStoragePeerClient(conn)
	return nil
}

func StartEndorserPeerClient(address string) error {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return err
	}
	endorserClients = append(endorserClients, NewStoragePeerClient(conn))
	return nil
}

func GetStoragePeerClient() StoragePeerClient{
	return storageClient
}

func GetEndorserPeerClients() []StoragePeerClient{
	return endorserClients
}
