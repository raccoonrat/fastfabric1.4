package remote

import (
	"google.golang.org/grpc"
)

var clients = make([]StoragePeerClient,0)

func StartStoragePeerClient(address string) error {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return err
	}
	clients = append(clients, NewStoragePeerClient(conn))
	return nil
}

func GetStoragePeerClients() []StoragePeerClient{
	return clients
}
