package gossip

import (
	"fmt"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/fastfabric-extensions/unmarshaled"
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
	VerifyUnmarshaledBlock(chainID common.ChainID, uBlock *unmarshaled.Block) error
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

func (m *MSPMessageCryptoService) VerifyUnmarshaledBlock(chainID common.ChainID,  block *unmarshaled.Block) error {
	if block.ChannelId != string(chainID) {
		return fmt.Errorf("Invalid block's channel id. Expected [%s]. Given [%s]", chainID, block.ChannelId)
	}

	// - Get Policy for block validation

	// Get the policy manager for channelID
	cpm, ok := m.channelPolicyManagerGetter.Manager(block.ChannelId)
	if cpm == nil {
		return fmt.Errorf("Could not acquire policy manager for channel %s", block.ChannelId)
	}
	// ok is true if it was the manager requested, or false if it is the default manager
	mcsLogger.Debugf("Got policy manager for channel [%s] with flag [%t]", block.ChannelId, ok)

	// Get block validation policy
	policy, ok := cpm.GetPolicy(policies.BlockValidation)
	// ok is true if it was the policy requested, or false if it is the default policy
	mcsLogger.Debugf("Got block validation policy for channel [%s] with flag [%t]", block.ChannelId, ok)

	// - Prepare SignedData
	signatureSet := []*pcommon.SignedData{}
	for i, sig := range block.Metadata.Signatures{
		signatureSet = append(
			signatureSet,
			&pcommon.SignedData{
				Identity:  sig.SignatureHeader.Creator,
				Data:      util.ConcatenateBytes(block.Metadata.Value, block.Metadata.Signatures[i].MetadataSignature.SignatureHeader, block.Header.Bytes()),
				Signature: sig.Signature,
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