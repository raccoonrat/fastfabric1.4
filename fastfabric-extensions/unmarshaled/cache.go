package unmarshaled

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/chaincode/platforms/golang"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
)

type Block struct {
	Raw      *common.Block
	Header   *common.BlockHeader
	data     *Transactions
	metadata *BlockMetadata
}

type Transactions struct {
	*common.BlockData
	unmarshaled []bool
	txs	[]*Tx
}

type BlockMetadata struct {
	Raw         *common.BlockMetadata
	Metadata    []*Metadata
	initialized []bool
}

type Metadata struct {
	Signatures []*MetadataSignature
	Raw        *common.Metadata
}

type MetadataSignature struct {
	*common.MetadataSignature
	signatureHeader  *common.SignatureHeader
}

func GetBlock(raw *common.Block) (*Block, error) {
	if raw == nil {
		return nil, fmt.Errorf("Block must not be nil.")
	}
	if raw.Header == nil {
		return nil, fmt.Errorf("Block [%d] header is nil.", raw.Header)
	}

	if !bytes.Equal(raw.Data.Hash(), raw.Header.DataHash ) {
		return nil, fmt.Errorf("Header.DataHash is different from Hash(block.Data) for block with id [%d]", raw.Header.Number)
	}

	return &Block{
		Raw: raw,
		Header: raw.Header}, nil
}

func (b *Block) UnmarshalAllMetadata() (*BlockMetadata, error) {
	if b.metadata == nil{
		b.metadata = newBlockMetadata(b.Raw.Metadata)
	}
	for i, m := range b.metadata.Metadata {
		if m == nil {
			um, e := b.UnmarshalSpecificMetadata(common.BlockMetadataIndex(i))
			if e != nil{
				return nil, e
			}
			for _, s:=range um.Signatures {
				_, e = s.UnmarshalSignatureHeader()
				if e != nil{
					return nil, e
				}
			}
		}
	}
	return b.metadata, nil
}

func (b *Block) UnmarshalSpecificMetadata(index common.BlockMetadataIndex) (*Metadata, error) {
	if b.metadata.initialized[index]{
		return b.metadata.Metadata[index], nil
	}
	md := &common.Metadata{}
	err := proto.Unmarshal(b.Raw.Metadata.Metadata[index], md)
	if err != nil {
		return nil, errors.Wrapf(err, "error unmarshaling metadata from block at index [%s]", index)
	}
	newMd := newMetadata(md)
	b.metadata.Metadata[index] = newMd
	return newMd, nil
}

func (b *Block) GetChannelId()(string, error) {
	tx, err := b.data.unmarshalSpecificTx(0)
	if err != nil {
		return "", err
	}
	return tx.GetChannelId()
}

func (b *Block) UnmarshalAll() error {
	panic("implement me")
}


func (sig *MetadataSignature) UnmarshalSignatureHeader() (*common.SignatureHeader,error) {
	if sig.signatureHeader != nil {
		return sig.signatureHeader, nil
	}

	sh := &common.SignatureHeader{}
	err := proto.Unmarshal(sig.SignatureHeader, sh)
	if err != nil {
		return nil, errors.Wrap(err, "error unmarshaling SignatureHeader")
	}
	sig.signatureHeader = sh
	return sh, nil
}

func (block *Block) UnmarshalTransactions() ([]*Tx, error){
	if block.data == nil {
		block.data = newData(block.Raw.Data)
	}
	for i, _ := range block.data.Data {
		_, err := block.data.unmarshalSpecificTx(i)
		if err != nil {
			return nil, err
		}
	}

	return block.data.txs, nil
}

func (data Transactions) unmarshalSpecificTx(index int) (*Tx, error) {
	if data.txs[index] != nil {
		return data.txs[index], nil
	}

	panic("implement me")
}

func newData(data *common.BlockData) *Transactions {
	return &Transactions{
		BlockData:data,
		txs: make([]*Tx, len(data.Data)),
		unmarshaled:make([]bool, len(data.Data))}
}

func newBlockMetadata(metadata *common.BlockMetadata) *BlockMetadata {
	return &BlockMetadata{
		Raw: metadata,
		Metadata:make([]*Metadata, len(metadata.Metadata)),
		initialized:make([]bool, len(metadata.Metadata))}
}

func newMetadata(metadata *common.Metadata) *Metadata {
	md := &Metadata{
		Raw:        metadata,
		Signatures: make([]*MetadataSignature, len(metadata.Signatures)),
	}

	for i, s := range metadata.Signatures {
		md.Signatures[i] = &MetadataSignature{MetadataSignature:s}
	}
	return md
}

func NewHeader(hdr *common.Header) (*Header, error){
	utils.UnmarshalChannelHeader(hdr.ChannelHeader)

	if hdr == nil {
		return nil, fmt.Errorf("Header is nil.")
	}

	header := &Header{Raw:hdr,
		ChannelHeader:&ChannelHeader{
			Extension:&ChaincodeHeaderExtension{}},
		SignatureHeader:&SignatureHeader{}}

	if hdr.ChannelHeader == nil {
		return header, fmt.Errorf("ChannelHeader is nil.")
	}
	header.ChannelHeader.ChannelHeader, header.ChannelHeader.Err = utils.UnmarshalChannelHeader(hdr.ChannelHeader)
	if header.ChannelHeader.Err != nil{
		return header, header.ChannelHeader.Err
	}

	che:= header.ChannelHeader.Extension
	che.Err  = proto.Unmarshal(header.ChannelHeader.ChannelHeader.Extension, che)
	header.ChannelHeader.Extension = che
	if che.Err != nil {
		return header, che.Err
	}


	if hdr.SignatureHeader == nil {
		return header, fmt.Errorf("SignatureHeader is nil.")
	}
	header.SignatureHeader.SignatureHeader, header.SignatureHeader.Err = utils.GetSignatureHeader(hdr.SignatureHeader)
	if header.SignatureHeader.Err != nil{
		return header, header.SignatureHeader.Err
	}

	return header, nil
}

type ChaincodeEndorsedAction struct {
	*peer.ChaincodeEndorsedAction
	ProposalResponsePayload *ProposalResponsePayload
}

type TransactionAction struct {
	Raw     *peer.TransactionAction
	Header  *SignatureHeader
	Payload *ChaincodeActionPayload
}

type Transaction struct {
	Raw *peer.Transaction
	Err error
	Actions []*TransactionAction
}

func unmarshalTx(data []byte) *Tx {
	if data == nil {
		return nil
	}

	tx := &Tx{
		Data:data,
		Envelope:&Envelope{
			Payload:&Payload{
				Transaction:&Transaction{}}}}
	if tx.Envelope.Raw, tx.Envelope.Err= utils.GetEnvelopeFromBlock(data); tx.Envelope.Err != nil{
		return tx
	}

	env := tx.Envelope
	if env.Payload.Raw, env.Payload.Err = utils.GetPayload(tx.Envelope.Raw); env.Payload.Err != nil{
		return tx
	}

	pl := env.Payload
	var err error
	pl.Header, err = NewHeader(pl.Raw.Header)
	if err != nil{
		return tx
	}
	tx.channelId = pl.Header.ChannelHeader.ChannelId

	pl.Transaction.Raw, pl.Transaction.Err = utils.GetTransaction(pl.Raw.Data)
	if pl.Transaction.Err != nil{
		return tx
	}

	pl.Transaction.Actions = unmarshalActions(pl.Transaction)

	return tx
}

type DeploymentSpec struct {
	*peer.ChaincodeDeploymentSpec
	Err error
}

type ChaincodeInvocationSpec struct {
	*peer.ChaincodeInvocationSpec
	DeploymentSpec *DeploymentSpec
	Err error
}

type ChaincodeProposalPayload struct {
	*peer.ChaincodeProposalPayload
	Err error
	Input *ChaincodeInvocationSpec
}

type ChaincodeActionPayload struct {
	Raw *peer.ChaincodeActionPayload
	Err error
	Action	*ChaincodeEndorsedAction
	ChaincodeProposalPayload *ChaincodeProposalPayload
}

type TxRwSet struct {
	*rwsetutil.TxRwSet
	Err error
}

type ChaincodeEvent struct {
	*peer.ChaincodeEvent
	Err error
}

type ChaincodeAction struct {
	*peer.ChaincodeAction
	Err error
	Results *TxRwSet
	Events	*ChaincodeEvent
}

type ProposalResponsePayload struct {
	*peer.ProposalResponsePayload
	Err       error
	Extension *ChaincodeAction
}

func unmarshalActions(tx *Transaction) []*TransactionAction {
	if tx.Raw.Actions == nil{
		return make([]*TransactionAction, 0)
	}
	actions := make([]*TransactionAction, len(tx.Raw.Actions))
	for i, a := range tx.Raw.Actions {
		action := &TransactionAction{
			Raw: a,
			Header: &SignatureHeader{}}

		if action.Raw.Header != nil {
			action.Header.SignatureHeader, action.Header.Err = utils.GetSignatureHeader(action.Raw.Header)
		}

		action.Payload = unmarshalChaincodeActionPayload(action)

		actions[i] = action
	}

	return actions
}

func unmarshalChaincodeActionPayload(action *TransactionAction) (*ChaincodeActionPayload) {
	payload := &ChaincodeActionPayload{
		ChaincodeProposalPayload:&ChaincodeProposalPayload{
			Input:&ChaincodeInvocationSpec{
				DeploymentSpec:&DeploymentSpec{}}}}
	payload.Raw, payload.Err = utils.GetChaincodeActionPayload(action.Raw.Payload)
	if payload.Err != nil {
		return payload
	}
	payload.Action = unmarshalChaincodeEndorsedAction(payload)

	payload.ChaincodeProposalPayload.ChaincodeProposalPayload, payload.ChaincodeProposalPayload.Err =
		utils.GetChaincodeProposalPayload(payload.Raw.ChaincodeProposalPayload)

	cpp := payload.ChaincodeProposalPayload
	cpp.Input.ChaincodeInvocationSpec = &peer.ChaincodeInvocationSpec{}
	cpp.Input.Err = proto.Unmarshal(cpp.ChaincodeProposalPayload.Input, cpp.Input.ChaincodeInvocationSpec)

	ccspec := cpp.Input.ChaincodeSpec
	if ccspec != nil && ccspec.Input != nil && ccspec.Input.Args!= nil && len(ccspec.Input.Args) > 1 {
		cpp.Input.DeploymentSpec.ChaincodeDeploymentSpec, cpp.Input.DeploymentSpec.Err =
			utils.GetChaincodeDeploymentSpec(cpp.Input.ChaincodeSpec.Input.Args[2],platforms.NewRegistry(&golang.Platform{}))
	}

	return payload
}

func unmarshalChaincodeEndorsedAction(payload *ChaincodeActionPayload) *ChaincodeEndorsedAction  {
	action := &ChaincodeEndorsedAction{
		ProposalResponsePayload:&ProposalResponsePayload{
			Extension:&ChaincodeAction{
				Results:&TxRwSet{},
				Events:&ChaincodeEvent{}}}}
	ccAction := payload.Action
	ccAction.ProposalResponsePayload.ProposalResponsePayload, ccAction.ProposalResponsePayload.Err =
		utils.GetProposalResponsePayload(ccAction.ChaincodeEndorsedAction.ProposalResponsePayload)
	if ccAction.ProposalResponsePayload.Err != nil {
		return action
	}
	respExt := ccAction.ProposalResponsePayload.Extension
	respExt.ChaincodeAction, respExt.Err = utils.GetChaincodeAction(ccAction.ProposalResponsePayload.ProposalResponsePayload.Extension)
	if respExt.Err != nil {
		return action
	}

	respExt.Results.Err = respExt.Results.TxRwSet.FromProtoBytes(respExt.ChaincodeAction.Results)
	respExt.Events.Err = proto.Unmarshal(respExt.ChaincodeAction.Events, respExt.Events.ChaincodeEvent)

	return action
}

type Tx struct {
	Data []byte
	Envelope *Envelope
}

func (tx *Tx) GetChannelId() (string, error) {
	tx.Envelope.Payload.Header.ChannelHeader.Unmarshal()
	return tx.Envelope.Payload.Header.ChannelHeader.ChannelId, tx.Envelope.Payload.Header.ChannelHeader.Err
}

type Envelope struct {
	Raw *common.Envelope
	Err error
	Payload *Payload
}

type ChaincodeHeaderExtension struct {
	*peer.ChaincodeHeaderExtension
	Err error
}

type ChannelHeader struct {
	*common.ChannelHeader
	Extension	*ChaincodeHeaderExtension
	Err	error
	ChannelId string
}

func (header *ChannelHeader) Unmarshal() {

}

type SignatureHeader struct {
	*common.SignatureHeader
	Err error
}

type Header struct {
	Raw *common.Header
	ChannelHeader *ChannelHeader
	SignatureHeader *SignatureHeader

}

type Payload struct {
	Raw *common.Payload
	Err	error
	Header *Header
	Transaction *Transaction
}
