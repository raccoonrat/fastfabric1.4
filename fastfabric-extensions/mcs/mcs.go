package mcs

import (
	"fmt"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/fastfabric-extensions/unmarshaling"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/peer/gossip"
	pcommon "github.com/hyperledger/fabric/protos/common"
	"time"
)
var mcsLogger = flogging.MustGetLogger("fastfabric-extensions.mcs")

type MessageCryptoService interface {
	api.MessageCryptoService

	// VerifyBlock returns nil if the block is properly signed, and the claimed seqNum is the
	// sequence number that the block's header contains.
	// else returns error
	VerifyBlockByNum(chainID common.ChainID, seqNum uint64) error
}

type MSPMessageCryptoService struct {
	service api.MessageCryptoService
	channelPolicyManagerGetter policies.ChannelPolicyManagerGetter
	localSigner                crypto.LocalSigner
	deserializer               mgmt.DeserializersManager
}

func (m *MSPMessageCryptoService) GetPKIidOfCert(peerIdentity api.PeerIdentityType) common.PKIidType {
	return m.service.GetPKIidOfCert(peerIdentity)
}

func (m *MSPMessageCryptoService) VerifyBlock(chainID common.ChainID, seqNum uint64, signedBlock []byte) error {
	return m.service.VerifyBlock(chainID, seqNum, signedBlock)
}

func (m *MSPMessageCryptoService) Sign(msg []byte) ([]byte, error) {
	return m.service.Sign(msg)
}

func (m *MSPMessageCryptoService) Verify(peerIdentity api.PeerIdentityType, signature, message []byte) error {
	return m.Verify(peerIdentity,signature,message)
}

func (m *MSPMessageCryptoService) VerifyByChannel(chainID common.ChainID, peerIdentity api.PeerIdentityType, signature, message []byte) error {
	return m.VerifyByChannel(chainID, peerIdentity, signature, message)
}

func (m *MSPMessageCryptoService) ValidateIdentity(peerIdentity api.PeerIdentityType) error {
	return m.service.ValidateIdentity(peerIdentity)
}

func (m *MSPMessageCryptoService) Expiration(peerIdentity api.PeerIdentityType) (time.Time, error) {
	return m.service.Expiration(peerIdentity)
}

func (m *MSPMessageCryptoService) VerifyBlockByNum(chainID common.ChainID, seqNum uint64) error {
	// - Convert signedBlock to common.Block.
	block, err := unmarshaling.Cache.Get(seqNum)
	if err != nil {
		return fmt.Errorf("Failed unmarshalling block [%d] bytes on channel [%s]: [%s]", seqNum,  chainID, err)
	}

	if block.Raw.Header == nil {
		return fmt.Errorf("Invalid Block on channel [%s]. Header must be different from nil.", chainID)
	}

	// - Extract channelID and compare with chainID
	channelID, err := block.GetChannelId()
	if err != nil {
		return fmt.Errorf("Failed getting channel id from block with id [%d] on channel [%s]: [%s]", block.Raw.Header.Number, chainID, err)
	}

	if channelID != string(chainID) {
		return fmt.Errorf("Invalid block's channel id. Expected [%s]. Given [%s]", chainID, channelID)
	}

	metadata, err := block.GetMetadata()
	if err != nil {
		return fmt.Errorf("Failed unmarshalling medatata for signatures [%s]", err)
	}

	// - Verify that Header.DataHash is equal to the hash of block.Data
	// This is to ensure that the header is consistent with the data carried by this block
	if err = block.VerifyHash(); err != nil {
		return err
	}

	// - Get Policy for block validation

	// Get the policy manager for channelID
	cpm, ok := m.channelPolicyManagerGetter.Manager(channelID)
	if cpm == nil {
		return fmt.Errorf("Could not acquire policy manager for channel %s", channelID)
	}
	// ok is true if it was the manager requested, or false if it is the default manager
	mcsLogger.Debugf("Got policy manager for channel [%s] with flag [%t]", channelID, ok)

	// Get block validation policy
	policy, ok := cpm.GetPolicy(policies.BlockValidation)
	// ok is true if it was the policy requested, or false if it is the default policy
	mcsLogger.Debugf("Got block validation policy for channel [%s] with flag [%t]", channelID, ok)

	// - Prepare SignedData
	signatureSet := []*pcommon.SignedData{}
	signatureHeaders, err := block.GetSignatureHeaders()
	if err != nil {
		return fmt.Errorf("Failed unmarshalling signature header for block with id [%d] on channel [%s]: [%s]", block.Raw.Header.Number, chainID, err)
	}
	for i, shdr := range signatureHeaders{
		signatureSet = append(
			signatureSet,
			&pcommon.SignedData{
				Identity:  shdr.Creator,
				Data:      util.ConcatenateBytes(metadata.Value, metadata.Signatures[i].SignatureHeader, block.Raw.Header.Bytes()),
				Signature: metadata.Signatures[i].Signature,
			},
		)
	}

	// - Evaluate policy
	return policy.Evaluate(signatureSet)
}

func NewMSPMessageCryptoService(channelPolicyManagerGetter policies.ChannelPolicyManagerGetter, localSigner crypto.LocalSigner, deserializer mgmt.DeserializersManager)(*MSPMessageCryptoService){
	return &MSPMessageCryptoService{
		service:gossip.NewMCS(channelPolicyManagerGetter, localSigner, deserializer),
		channelPolicyManagerGetter:channelPolicyManagerGetter,
		localSigner:localSigner,
		deserializer:deserializer}
}