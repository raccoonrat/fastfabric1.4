package cached

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

func GetBlock(raw *common.Block) (*Block, error) {
	if raw == nil {
		return nil, fmt.Errorf("block must not be nil")
	}

	lenMeta := 0
	if raw.Metadata != nil {
		lenMeta = len(raw.Metadata.Metadata)
	}

	lenEnvs := 0
	if raw.Data != nil {
		lenEnvs = len(raw.Data.Data)
	}

	return &Block{
		Block: raw,
		cachedMetadata: make([]*Metadata, lenMeta),
		cachedEnvs: make([]*Envelope, lenEnvs)},
	nil
}

func (b *Block) UnmarshalAll() error {
	metas, err := b.UnmarshalAllMetadata()
	if err != nil{
		return err
	}
	for _, meta := range metas{
		if _, err :=meta.UnmarshalAllSignatureHeaders(); err != nil{
			return err
		}
	}
	envs, err :=b.UnmarshalData();
	if err != nil{
		return err
	}

	for _, env := range envs{
		pl, err := env.UnmarshalPayload();
		if err != nil{
			return err
		}
		chdr, err := pl.UnmarshalChannelHeader()
		if err != nil{
			return err
		}
		_, err = chdr.UnmarshalExtension()
		if err != nil{
			return err
		}

		_, err = pl.UnmarshalSignatureHeader()
		if err != nil{
			return err
		}

		etx, err :=pl.UnmarshalEndorserTransaction()
		if err != nil{
			return err
		}

		for _,act := range etx.Actions {
			_, err := act.UnmarshalHeader()
			if err != nil{
				return err
			}
			pl, err := act.UnmarshalChaincodeActionPayload()
			if err != nil{
				return err
			}

			respPl, err := pl.Action.UnmarshalProposalResponsePayload()
			if err != nil{
				return err
			}

			act, err := respPl.UnmarshalChaincodeAction()
			if err != nil{
				return err
			}

			_, err = act.UnmarshalRwSet()
			if err != nil{
				return err
			}

			_, err = act.UnmarshalEvents()
			if err != nil{
				return err
			}

			propPl, err := pl.UnmarshalProposalPayload()
			if err != nil{
				return err
			}

			input, err := propPl.UnmarshalInput()
			if err != nil{
				return err
			}
			input.
		}
	}
	return nil
}

func (b *Block) UnmarshalAllMetadata() ([]*Metadata, error) {
	if b.Metadata == nil {
		return nil, fmt.Errorf("block metadata must not be nil")
	}

	for i  := range b.Metadata.Metadata {
		if _, err := b.UnmarshalSpecificMetadata(common.BlockMetadataIndex(i)); err != nil{
			return nil, err
		}
	}
	return b.cachedMetadata, nil
}

func (b *Block) UnmarshalSpecificMetadata(index common.BlockMetadataIndex) (*Metadata, error) {
	if len(b.cachedMetadata)  <= int(index) || index < 0 {
		return nil, fmt.Errorf("index out of range")
	}

	if b.cachedMetadata[index] != nil{
		return b.cachedMetadata[index], nil
	}

	metaRaw := &common.Metadata{}
	if err := proto.Unmarshal(b.Metadata.Metadata[index], metaRaw); err != nil {
		return nil, errors.Wrapf(err, "error unmarshaling metadata from block at index [%s]", index)
	}

	meta := &Metadata{
		Metadata:metaRaw,
		cachedSigHeaders:make([]*common.SignatureHeader, len(metaRaw.Signatures))}
	b.cachedMetadata[index] = meta
	return meta, nil
}

func (meta *Metadata) UnmarshalSpecificSignatureHeader(index int) (*common.SignatureHeader,error) {
	if len(meta.cachedSigHeaders)  <= int(index) || index < 0 {
		return nil, fmt.Errorf("index out of range")
	}
	if meta.cachedSigHeaders[index] != nil {
		return meta.cachedSigHeaders[index], nil
	}

	var err error
	meta.cachedSigHeaders[index], err = unmarshalSignatureHeader(meta.Signatures[index].SignatureHeader)
	return meta.cachedSigHeaders[index], err
}

func (meta *Metadata) UnmarshalAllSignatureHeaders() ([]*common.SignatureHeader, error) {
	for i := range meta.Signatures {
		_, err := meta.UnmarshalSpecificSignatureHeader(i)
		if err != nil {
			return nil, err
		}
	}
	return meta.cachedSigHeaders, nil
}

func (b *Block) UnmarshalData() ([]*Envelope, error){
	if b.Data == nil || b.Data.Data == nil {
		return nil, fmt.Errorf("block data must not be nil")
	}

	for i := range b.Data.Data {
		if _, err := b.unmarshalSpecificData(i); err != nil {
			return nil, err
		}
	}

	return b.cachedEnvs, nil
}

func (b Block) unmarshalSpecificData(index int) (*Envelope, error) {
	if b.Data == nil || b.Data.Data == nil {
		return nil, fmt.Errorf("block data must not be nil")
	}
	if len(b.cachedEnvs)  <= int(index) || index < 0 {
		return nil, fmt.Errorf("index out of range")
	}
	if b.cachedEnvs[index] != nil {
		return b.cachedEnvs[index], nil
	}

	envRaw := &common.Envelope{}
	if err := proto.Unmarshal(b.Data.Data[index], envRaw); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling Envelope")
	}

	env := &Envelope{Envelope: envRaw}
	b.cachedEnvs[index] = env
	return env, nil
}

func (env *Envelope) UnmarshalPayload() (*Payload, error) {
	if env.cachedPayload != nil {
		return env.cachedPayload, nil
	}

	payloadRaw := &common.Payload{}
	if err := proto.Unmarshal(env.Payload, payloadRaw); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling Payload")
	}

	payload := &Payload{Payload: payloadRaw}
	env.cachedPayload = payload
	return payload, nil
}

func(pl *Payload) UnmarshalChannelHeader() (*ChannelHeader, error) {
	if pl.cachedChanHeader != nil {
		return pl.cachedChanHeader, nil
	}

	if pl.Header == nil {
		return nil, fmt.Errorf("payload header is nil")
	}

	headerRaw := &common.ChannelHeader{}
	if err := proto.Unmarshal(pl.Header.ChannelHeader, headerRaw); err != nil{
		return nil, errors.Wrap(err, "error unmarshaling payload ChannelHeader")
	}

	header := &ChannelHeader{ChannelHeader: headerRaw}
	pl.cachedChanHeader = header
	return header, nil
}

func(pl *Payload) UnmarshalSignatureHeader() (*common.SignatureHeader, error) {
	if pl.cachedChanHeader != nil {
		return pl.cachedSigHeader, nil
	}

	if pl.Header == nil {
		return nil, fmt.Errorf("payload header is nil")
	}

	headerRaw, err := unmarshalSignatureHeader(pl.Header.SignatureHeader)

	pl.cachedSigHeader = headerRaw
	return headerRaw, err
}

func unmarshalSignatureHeader(bytes []byte) (*common.SignatureHeader, error){
	headerRaw := &common.SignatureHeader{}
	if err := proto.Unmarshal(bytes, headerRaw); err != nil{
		return nil, errors.Wrap(err, "error unmarshaling SignatureHeader")
	}
	return headerRaw, nil
}

func (ch *ChannelHeader) UnmarshalExtension() (*peer.ChaincodeHeaderExtension, error){
	if ch.cachedExtension != nil{
		return ch.cachedExtension, nil
	}

	ext := &peer.ChaincodeHeaderExtension{}
	if err := proto.Unmarshal(ch.Extension, ext); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling channel header ChaincodeHeaderExtension")
	}
	ch.cachedExtension = ext
	return ext, nil
}

func (pl *Payload) UnmarshalEndorserTransaction() (*Transaction, error) {
	if pl.cachedEnTx != nil{
		return pl.cachedEnTx, nil
	}
	txRaw := &peer.Transaction{}
	if err := proto.Unmarshal(pl.Data, txRaw); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling Transaction")
	}

	tx := &Transaction{Transaction: txRaw, Actions: make([]*TransactionAction, len(txRaw.Actions))}
	for i, a := range tx.Actions{
		tx.Actions[i] = a
	}
	pl.cachedEnTx = tx
	return tx, nil
}

func (act *TransactionAction) UnmarshalHeader() (*common.SignatureHeader, error) {
	if act.cachedSigHeader != nil {
		return act.cachedSigHeader, nil
	}

	var err error
	act.cachedSigHeader, err = unmarshalSignatureHeader(act.Header)
	return act.cachedSigHeader, err
}

func (act *TransactionAction) UnmarshalChaincodeActionPayload() (*ChaincodeActionPayload, error) {
	if act.cachedActionPayload != nil {
		return act.cachedActionPayload, nil
	}

	capRaw := &peer.ChaincodeActionPayload{}
	if err := proto.Unmarshal(act.Payload, capRaw); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling ChaincodeActionPayload")
	}

	act.cachedActionPayload = &ChaincodeActionPayload{
		ChaincodeActionPayload: capRaw,
		Action:&ChaincodeEndorsedAction{
			ChaincodeEndorsedAction:capRaw.Action}}
	return act.cachedActionPayload, nil
}

func (pl *ChaincodeActionPayload) UnmarshalProposalPayload()(*ChaincodeProposalPayload, error) {
	if pl.cachedPropPayload != nil {
		return pl.cachedPropPayload, nil
	}

	cpp := &peer.ChaincodeProposalPayload{}
	if err := proto.Unmarshal(pl.ChaincodeProposalPayload, cpp); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling ChaincodeProposalPayload")
	}

	pl.cachedPropPayload = &ChaincodeProposalPayload{ChaincodeProposalPayload:cpp}
	return pl.cachedPropPayload, nil
}

func (act *ChaincodeEndorsedAction) UnmarshalProposalResponsePayload() (*ProposalResponsePayload, error) {
	if act.cachedRespPayload != nil {
		return act.cachedRespPayload, nil
	}

	prp := &peer.ProposalResponsePayload{}
	if err := proto.Unmarshal(act.ProposalResponsePayload, prp); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling ProposalResponsePayload")
	}

	act.cachedRespPayload = &ProposalResponsePayload{ProposalResponsePayload: prp}
	return act.cachedRespPayload, nil
}

func (respPl *ProposalResponsePayload) UnmarshalChaincodeAction() (*ChaincodeAction, error){
	if respPl.cachedAction != nil {
		return respPl.cachedAction, nil
	}

	chaincodeAction := &peer.ChaincodeAction{}
	if err := proto.Unmarshal(respPl.Extension, chaincodeAction); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling ChaincodeAction")
	}

	respPl.cachedAction = &ChaincodeAction{ChaincodeAction:chaincodeAction}
	return respPl.cachedAction, nil
}

func (act *ChaincodeAction) UnmarshalRwSet()(*rwsetutil.TxRwSet, error) {
	if act.cachedRwSet != nil {
		return act.cachedRwSet, nil
	}

	rwset := &rwsetutil.TxRwSet{}
	if err := rwset.FromProtoBytes(act.Results); err != nil{
		return nil, err
	}

	act.cachedRwSet = rwset
	return rwset, nil
}

func (act *ChaincodeAction) UnmarshalEvents()(*peer.ChaincodeEvent, error) {
	if act.cachedEvents != nil {
		return act.cachedEvents, nil
	}
	event := &peer.ChaincodeEvent{}
	if err := proto.Unmarshal(act.Events, event); err != nil {
		return nil, err
	}

	act.cachedEvents = event
	return event, nil
}

func (cpp *ChaincodeProposalPayload) UnmarshalInput() (*ChaincodeInvocationSpec, error){
	if cpp.cachedInput != nil {
		return cpp.cachedInput, nil
	}

	cis := &peer.ChaincodeInvocationSpec{}
	if err := proto.Unmarshal(cpp.Input, cis); err != nil{
		return nil, err
	}
	cpp.cachedInput = &ChaincodeInvocationSpec{ChaincodeInvocationSpec:cis}
	return cpp.cachedInput, nil
}

ccspec := cpp.Input.ChaincodeSpec
if ccspec != nil && ccspec.Input != nil && ccspec.Input.Args!= nil && len(ccspec.Input.Args) > 1 {
cpp.Input.DeploymentSpec.ChaincodeDeploymentSpec, cpp.Input.DeploymentSpec.Err =
utils.GetChaincodeDeploymentSpec(cpp.Input.ChaincodeSpec.Input.Args[2],platforms.NewRegistry(&golang.Platform{}))
}
