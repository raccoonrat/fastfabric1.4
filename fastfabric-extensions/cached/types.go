package cached

import (
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
)

type Block struct {
	*common.Block
	cachedEnvs	[]*Envelope
	cachedMetadata []*Metadata
}

type Metadata struct {
	*common.Metadata
	cachedSigHeaders []*common.SignatureHeader
}

type Envelope struct {
	*common.Envelope
	cachedPayload *Payload
}

type Payload struct {
	*common.Payload
	cachedEnTx *Transaction
	Header     *Header
}

type ChannelHeader struct {
	*common.ChannelHeader
	cachedExtension	*peer.ChaincodeHeaderExtension
}

type Transaction struct {
	*peer.Transaction
	Actions []*TransactionAction
}

type TransactionAction struct {
	*peer.TransactionAction
	cachedSigHeader  *common.SignatureHeader
	cachedActionPayload *ChaincodeActionPayload
}

type ProposalResponsePayload struct {
	*peer.ProposalResponsePayload
	cachedAction *ChaincodeAction
}

type ChaincodeInvocationSpec struct {
	*peer.ChaincodeInvocationSpec
}

type ChaincodeProposalPayload struct {
	*peer.ChaincodeProposalPayload
	cachedInput *ChaincodeInvocationSpec
}

type ChaincodeActionPayload struct {
	*peer.ChaincodeActionPayload
	Action	*ChaincodeEndorsedAction
	cachedPropPayload *ChaincodeProposalPayload
}

type ChaincodeEndorsedAction struct {
	*peer.ChaincodeEndorsedAction
	cachedRespPayload *ProposalResponsePayload
}


type ChaincodeAction struct {
	*peer.ChaincodeAction
	cachedRwSet  *TxRwSet
	cachedEvents *peer.ChaincodeEvent
}

type Header struct{
	*common.Header
	cachedChanHeader *ChannelHeader
	cachedSigHeader  *common.SignatureHeader
}
