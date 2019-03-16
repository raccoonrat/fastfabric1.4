package grpcmocks

import (
	"bytes"
	"context"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

var CrClient CryptoClientMock
var StClient StorageClient

type CryptoClientMock struct {
	client *CryptoClient
}

type StorageClientMock struct {
	client *StorageClient
}

func (c* CryptoClientMock) Verify(ctx context.Context, tx *Transaction)(*Result, error){
	mspObj := mspmgmt.GetIdentityDeserializer(tx.ChainID)
	if mspObj == nil {
		return nil, errors.Errorf("could not get msp for channel [%s]", tx.ChainID)
	}

	// get the identity of the creator
	creator, err := mspObj.DeserializeIdentity(tx.Creator)
	if err != nil {

		return nil, errors.WithMessage(err, "MSP error")
	}

	// ensure that creator is a valid certificate
	err = creator.Validate()
	if err != nil {
		return nil, errors.WithMessage(err, "creator certificate is not valid")
	}


	// validate the signature
	err = creator.Verify(tx.Data, tx.Signature)
	if err != nil {
		return nil, errors.WithMessage(err, "creator's signature over the proposal is not valid")
	}

	return &Result{}, nil
}

func (c* CryptoClientMock) CompareHash(ctx context.Context, in *CompareMessage, opts ...grpc.CallOption) (*Result, error) {


	// build the original header by stitching together
	// the common ChannelHeader and the per-action SignatureHeader
	hdrOrig := &common.Header{ChannelHeader: in.ChannelHdr, SignatureHeader: in.ActionHdr}

	// compute proposalHash
	pHash, err := utils.GetProposalHash2(hdrOrig, in.ProposalPayload)
	if err != nil {
		return nil, err
	}

	// ensure that the proposal hash matches
	if bytes.Compare(pHash, in.ProposalHash) != 0 {
		return nil, errors.New("proposal hash does not match")
	}
	return &Result{}, nil
}

var conn grpc.ClientConn

func StartClients(address string) {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	cr := NewCryptoClient(conn)
	StClient = NewStorageClient(conn)
	CrClient = CryptoClientMock{client:&cr}
}

func StopClient() {
	conn.Close()
}
