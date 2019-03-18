package blocks

import (
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
)

type blockCache []*UnmarshaledBlock

const MAXCACHEINDEX = 1023

var Cache = make(blockCache, MAXCACHEINDEX+1)
var pvtData = make([]*ledger.BlockAndPvtData, MAXCACHEINDEX+1)

func (cache blockCache) Put(block *common.Block) error {
	cache[findCircularIndex(block.Header.Number)] = newBlock(block)
	return nil
}
func (cache blockCache) Get(blockNo uint64) (*UnmarshaledBlock, error) {
	block := cache[findCircularIndex(blockNo)]
	if block == nil {
		return nil, errors.New("blockNo not found")
	}
	if block.Rawblock.Header.Number != blockNo {
		return nil, errors.New(fmt.Sprintln("Found block", block.Rawblock.Header.Number, "instead"))
	}

	return block, nil
}

func GetPvtData(blockNo uint64) (*ledger.BlockAndPvtData, error) {
	data := pvtData[findCircularIndex(blockNo)]
	if data == nil || data.Block.Header.Number != blockNo {
		return nil, errors.New("blockNo not found")
	}

	return data, nil
}

type UnmarshaledBlock struct {
	Rawblock *common.Block
	meta     *common.Metadata
	metaErr  error
	sigs     []*common.SignatureHeader
	sigsErr  error
	Txs      []*UnmarshaledTx
}

type UnmarshaledTx struct {
	rawTx         []byte
	finished      bool
	envErr        error
	envelope      *common.Envelope
	channelHeader *common.ChannelHeader
	chErr         error
	shErr         error
	sigHeader     *common.SignatureHeader
	ext           *peer.ChaincodeHeaderExtension
	extErr        error
	payload       *common.Payload
	plErr         error

	raw     *peer.Transaction
	rawErr  error
	actions []*UnmarshalledAction
}

type UnmarshalledAction struct {
	Raw       *peer.TransactionAction
	sigHeader *common.SignatureHeader
	shErr     error
	payload   *peer.ChaincodeActionPayload
	action    *peer.ChaincodeAction
	acErr     error
	plErr     error
	rwSet     *rwsetutil.TxRwSet
	rwErr     error
	propPlErr error
	propPl    *peer.ChaincodeProposalPayload
	invErr    error
	invSpec   *peer.ChaincodeInvocationSpec
	depErr    error
	depSpec   *peer.ChaincodeDeploymentSpec
	evErr     error
	event     *peer.ChaincodeEvent
	respPlErr error
	respPl    *peer.ProposalResponsePayload
}

func findCircularIndex(blockNo uint64) uint64 {
	return blockNo & MAXCACHEINDEX
}

func newBlock(rawBlock *common.Block) *UnmarshaledBlock {
	b := &UnmarshaledBlock{Rawblock: rawBlock, Txs: make([]*UnmarshaledTx, len(rawBlock.Data.Data))}
	for idx, data := range b.Rawblock.Data.Data {
		newTx := &UnmarshaledTx{rawTx: data}
		b.Txs[idx] = newTx
	}
	return b
}

func (b *UnmarshaledBlock) GetMetadata() (*common.Metadata, error) {
	if b.meta == nil && b.metaErr == nil {
		b.meta, b.metaErr = utils.GetMetadataFromBlock(b.Rawblock, common.BlockMetadataIndex_SIGNATURES)
	}
	return b.meta, b.metaErr
}

func (b *UnmarshaledBlock) GetSignatureHeaders() ([]*common.SignatureHeader, error) {
	if b.sigs == nil && b.sigsErr == nil {
		var meta *common.Metadata
		if meta, b.sigsErr = b.GetMetadata(); b.sigsErr == nil {
			b.sigs = make([]*common.SignatureHeader, len(meta.Signatures))
			for i, metadataSignature := range meta.Signatures {
				if b.sigs[i], b.sigsErr = utils.GetSignatureHeader(metadataSignature.SignatureHeader); b.sigsErr != nil {
					break
				}
			}
		}
	}
	return b.sigs, b.sigsErr
}

func (t *UnmarshaledTx) GetEnv() (*common.Envelope, error) {
	if t.envelope == nil && t.envErr == nil {
		t.envelope, t.envErr = utils.GetEnvelopeFromBlock(t.rawTx)
	}
	return t.envelope, t.envErr
}

func (t *UnmarshaledTx) GetPayload() (*common.Payload, error) {
	if t.payload == nil && t.plErr == nil {
		var env *common.Envelope
		if env, t.plErr = t.GetEnv(); t.plErr == nil {
			t.payload, t.plErr = utils.GetPayload(env)
		}
	}
	return t.payload, t.plErr
}

func (t *UnmarshaledTx) GetPeerTransaction() (*peer.Transaction, error) {
	if t.raw == nil && t.rawErr == nil {
		var pl *common.Payload
		if pl, t.rawErr = t.GetPayload(); t.rawErr == nil {
			t.raw, t.rawErr = utils.GetTransaction(pl.Data)
		}
	}
	return t.raw, t.rawErr
}

func (t *UnmarshaledTx) GetChannelHeader() (*common.ChannelHeader, error) {
	if t.channelHeader == nil && t.chErr == nil {
		var pl *common.Payload
		if pl, t.chErr = t.GetPayload(); t.chErr == nil {
			if t.channelHeader, t.chErr = utils.UnmarshalChannelHeader(pl.Header.ChannelHeader); t.chErr == nil {
				t.ext = &peer.ChaincodeHeaderExtension{}
				t.extErr = proto.Unmarshal(t.channelHeader.Extension, t.ext)
			} else {
				t.extErr = t.chErr
			}
		}
	}
	return t.channelHeader, t.chErr
}

func (t *UnmarshaledTx) GetSignatureHeader() (*common.SignatureHeader, error) {
	if t.sigHeader == nil && t.shErr == nil {
		var pl *common.Payload
		if pl, t.shErr = t.GetPayload(); t.shErr == nil {
			t.sigHeader, t.shErr = utils.GetSignatureHeader(pl.Header.SignatureHeader)
		}
	}
	return t.sigHeader, t.shErr
}

func (t *UnmarshaledTx) GetExtension() (*peer.ChaincodeHeaderExtension, error) {
	return t.ext, t.extErr
}

func (t *UnmarshaledTx) GetActions() []*UnmarshalledAction {
	if t.actions == nil {
		if raw, err := t.GetPeerTransaction(); err == nil {
			t.actions = make([]*UnmarshalledAction, len(raw.Actions))
			for i := range t.actions {
				newAction := &UnmarshalledAction{Raw: raw.Actions[i]}
				t.actions[i] = newAction
			}
		}
	}
	return t.actions
}

func (act *UnmarshalledAction) GetActionPayload() (*peer.ChaincodeActionPayload, *peer.ChaincodeAction, error) {
	if act.payload == nil && act.action == nil && act.plErr == nil {
		act.payload, act.action, act.plErr = utils.GetPayloads(act.Raw)
	}
	return act.payload, act.action, act.plErr
}

func (act *UnmarshalledAction) GetEvent() (*peer.ChaincodeEvent, error) {
	if act.event == nil && act.evErr == nil {
		var pl *peer.ChaincodeAction
		if _, pl, act.evErr = act.GetActionPayload(); act.evErr == nil {
			act.event = &peer.ChaincodeEvent{}
			act.evErr = proto.Unmarshal(pl.Events, act.event)
		}
	}

	return act.event, act.evErr
}

func (act *UnmarshalledAction) GetProposalPayload() (*peer.ChaincodeProposalPayload, error) {
	if act.propPl == nil && act.propPlErr == nil {
		var pl *peer.ChaincodeActionPayload
		if pl, _, act.propPlErr = act.GetActionPayload(); act.propPlErr == nil {
			act.propPl, act.propPlErr = utils.GetChaincodeProposalPayload(pl.ChaincodeProposalPayload)
		}
	}
	return act.propPl, act.propPlErr
}

func (act *UnmarshalledAction) GetProposalResponsePayload() (*peer.ProposalResponsePayload, error) {
	if act.respPl == nil && act.respPlErr == nil {
		var pl *peer.ChaincodeActionPayload
		if pl, _, act.respPlErr = act.GetActionPayload(); act.respPlErr == nil {
			act.respPl, act.propPlErr = utils.GetProposalResponsePayload(pl.Action.ProposalResponsePayload)
		}
	}
	return act.respPl, act.respPlErr
}

func (act *UnmarshalledAction) GetInvokeSpec() (*peer.ChaincodeInvocationSpec, error) {
	if act.invSpec == nil && act.invErr == nil {
		var prop *peer.ChaincodeProposalPayload
		if prop, act.invErr = act.GetProposalPayload(); act.invErr == nil {
			cis := &peer.ChaincodeInvocationSpec{}
			act.invErr = proto.Unmarshal(prop.Input, cis)
		}
	}

	return act.invSpec, act.invErr
}

func (act *UnmarshalledAction) GetDeploymentSpec() (*peer.ChaincodeDeploymentSpec, error) {
	if act.depSpec == nil && act.depErr == nil {
		var inv *peer.ChaincodeInvocationSpec
		if inv, act.depErr = act.GetInvokeSpec(); act.depErr == nil {
			if inv != nil {
				act.depSpec, act.depErr = utils.GetChaincodeDeploymentSpec(inv.ChaincodeSpec.Input.Args[2])
			}
		}
	}

	return act.depSpec, act.depErr
}

func (act *UnmarshalledAction) GetSigHeader() (*common.SignatureHeader, error) {
	if act.sigHeader == nil && act.shErr == nil {
		act.sigHeader, act.shErr = utils.GetSignatureHeader(act.Raw.Header)
	}

	return act.sigHeader, act.shErr
}

func (act *UnmarshalledAction) GetRwSet() (*rwsetutil.TxRwSet, error) {
	if act.rwSet == nil && act.rwErr == nil {
		var action *peer.ChaincodeAction
		if _, action, act.rwErr = act.GetActionPayload(); act.rwErr == nil {
			txRWSet := &rwsetutil.TxRwSet{}
			act.rwErr = txRWSet.FromProtoBytes(action.Results)
			act.rwSet = txRWSet
		}
	}
	return act.rwSet, act.rwErr
}
