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
)

const MAXCACHEINDEX = 1023

type Block struct {
    Raw *common.Block
	ChannelId 	string
	sigs      	[]*common.SignatureHeader
	sigsErr   	error
	Txs       	[]*Tx
	Metadata    *Metadata
    Header		*common.BlockHeader
}

type Metadata struct{
	Raw *common.BlockMetadata
	*common.Metadata
	Signatures []*MetadataSignature
}

type MetadataSignature struct {
	*common.MetadataSignature
	SignatureHeader  *common.SignatureHeader
}

func NewBlock(raw *common.Block) (*Block, error) {
	if raw == nil || raw.Data == nil || raw.Data.Data == nil{
		return nil, fmt.Errorf("There is no content in this block.")
	}
	if raw.Header == nil {
		return nil, fmt.Errorf("Block [%d] header is nil.", raw.Header)
	}
	block := &Block{Raw:raw, Header:raw.Header}
	var err error
	metadata, err := utils.GetMetadataFromBlock(raw, common.BlockMetadataIndex_SIGNATURES);
	if err != nil{
		return nil, err
	}
	block.Metadata = &Metadata{Metadata: metadata}
	if metadata.Signatures == nil {
		return nil, fmt.Errorf("Block [%d] metadata signatures is nil.", raw.Header)
	}

	block.Metadata.Signatures = make([]*MetadataSignature,len(metadata.Signatures))
	for i,sig := range metadata.Signatures {
		block.Metadata.Signatures[i] = &MetadataSignature{MetadataSignature: sig}
		sh, err := utils.GetSignatureHeader(sig.SignatureHeader)
		if err != nil {
			return nil, err
		}
		block.Metadata.Signatures[i].SignatureHeader = sh
	}

	block.Txs = make([]*Tx, len(raw.Data.Data))
	for idx, data := range raw.Data.Data {
		block.Txs[idx] = unmarshalTx(data)
	}

	if len(block.Txs) != 0{
		block.ChannelId = block.Txs[0].ChannelId
	}

	if !bytes.Equal(raw.Data.Hash(), raw.Header.DataHash ) {
		return nil, fmt.Errorf("Header.DataHash is different from Hash(block.Data) for block with id [%d] on channel [%s]", raw.Header.Number, block.ChannelId)
	}

	return block, nil
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
	tx.ChannelId = pl.Header.ChannelHeader.ChannelId

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
	ChannelId     string
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
