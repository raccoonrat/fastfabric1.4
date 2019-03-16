package crypto

import (
	"bytes"
	"fmt"
	"github.com/fabric_extension/block_cache"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/localmsp"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/msp/mgmt"
	pcommon "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
)

var messageCryptoService *MspMessageCryptoService
var chainID string
type MspMessageCryptoService struct {
	channelPolicyManagerGetter policies.ChannelPolicyManagerGetter
	localSigner                crypto.LocalSigner
	deserializer               mgmt.DeserializersManager
}

func SetChainID(chID string){
	chainID = chID
}

func GetService() *MspMessageCryptoService{
	if messageCryptoService == nil{
		messageCryptoService = &MspMessageCryptoService{
		channelPolicyManagerGetter: peer.NewChannelPolicyManagerGetter(),
		localSigner: localmsp.NewSigner(),
		deserializer: mgmt.NewDeserializersManager()}
	}
	return messageCryptoService
}

// VerifyBlockByNo returns nil if the block is properly signed, and the claimed seqNum is the
// sequence number that the block's header contains.
// else returns error
func (s *MspMessageCryptoService) VerifyBlockByNo(seqNum uint64) error {
	// - Convert signedBlock to common.Block.
	block, err := blocks.Cache.Get(seqNum)
	if err != nil {
		return fmt.Errorf("Failed unmarshalling block bytes on channel [%s]: [%s]", chainID, err)
	}

	if block.Rawblock.Header == nil {
		return fmt.Errorf("Invalid Block on channel [%s]. Header must be different from nil.", chainID)
	}

	blockSeqNum := block.Rawblock.Header.Number
	if seqNum != blockSeqNum {
		return fmt.Errorf("Claimed seqNum is [%d] but actual seqNum inside block is [%d]", seqNum, blockSeqNum)
	}

	// - Extract channelID and compare with chainID
	chdr, err := block.Txs[0].GetChannelHeader()
	if err != nil {
		return fmt.Errorf("Failed getting channel id from block with id [%d] on channel [%s]: [%s]", block.Rawblock.Header.Number, chainID, err)
	}
	channelID := chdr.ChannelId

	if channelID != string(chainID) {
		return fmt.Errorf("Invalid block's channel id. Expected [%s]. Given [%s]", chainID, channelID)
	}

	// - Unmarshal medatada
	if block.Rawblock.Metadata == nil || len(block.Rawblock.Metadata.Metadata) == 0 {
		return fmt.Errorf("Block with id [%d] on channel [%s] does not have metadata. Block not valid.", block.Rawblock.Header.Number, chainID)
	}

	metadata, err := block.GetMetadata()
	if err != nil {
		return fmt.Errorf("Failed unmarshalling medatata for signatures [%s]", err)
	}

	// - Verify that Header.DataHash is equal to the hash of block.Data
	// This is to ensure that the header is consistent with the data carried by this block
	if !bytes.Equal(block.Rawblock.Data.Hash(), block.Rawblock.Header.DataHash) {
		return fmt.Errorf("Header.DataHash is different from Hash(block.Data) for block with id [%d] on channel [%s]", block.Rawblock.Header.Number, chainID)
	}

	// - Get Policy for block validation

	// Get the policy manager for channelID
	//cpm, _ := s.channelPolicyManagerGetter.Manager(channelID)
	//if cpm == nil {
	//	return fmt.Errorf("Could not acquire policy manager for channel %s", channelID)
	//}

	// Get block validation policy
	//policy, _ := cpm.GetPolicy(policies.BlockValidation)

	// - Prepare SignedData
	signatureSet := make([]*pcommon.SignedData, len(metadata.Signatures))
	sigs, err := block.GetSignatureHeaders()
	if err != nil {
		return fmt.Errorf("Failed unmarshalling signature header for block with id [%d] on channel [%s]: [%s]", block.Rawblock.Header.Number, chainID, err)
	}
	for i, shdr := range sigs {

		signatureSet[i] = &pcommon.SignedData{
				Identity:  shdr.Creator,
				Data:      util.ConcatenateBytes(metadata.Value, metadata.Signatures[i].SignatureHeader, block.Rawblock.Header.Bytes()),
				Signature: metadata.Signatures[i].Signature,
			}
	}

	// - Evaluate policy
	//return policy.Evaluate(signatureSet)
	return nil
}

func (s *MspMessageCryptoService) VerifyBlock(chainID common.ChainID, seqNum uint64, signedBlock []byte) error {
	// - Convert signedBlock to common.Block.
	block, err := utils.GetBlockFromBlockBytes(signedBlock)
	if err != nil {
		return fmt.Errorf("Failed unmarshalling block bytes on channel [%s]: [%s]", chainID, err)
	}

	if block.Header == nil {
		return fmt.Errorf("Invalid Block on channel [%s]. Header must be different from nil.", chainID)
	}

	blockSeqNum := block.Header.Number
	if seqNum != blockSeqNum {
		return fmt.Errorf("Claimed seqNum is [%d] but actual seqNum inside block is [%d]", seqNum, blockSeqNum)
	}

	// - Extract channelID and compare with chainID
	channelID, err := utils.GetChainIDFromBlock(block)
	if err != nil {
		return fmt.Errorf("Failed getting channel id from block with id [%d] on channel [%s]: [%s]", block.Header.Number, chainID, err)
	}

	if channelID != string(chainID) {
		return fmt.Errorf("Invalid block's channel id. Expected [%s]. Given [%s]", chainID, channelID)
	}

	// - Unmarshal medatada
	if block.Metadata == nil || len(block.Metadata.Metadata) == 0 {
		return fmt.Errorf("Block with id [%d] on channel [%s] does not have metadata. Block not valid.", block.Header.Number, chainID)
	}

	metadata, err := utils.GetMetadataFromBlock(block, pcommon.BlockMetadataIndex_SIGNATURES)
	if err != nil {
		return fmt.Errorf("Failed unmarshalling medatata for signatures [%s]", err)
	}

	// - Verify that Header.DataHash is equal to the hash of block.Data
	// This is to ensure that the header is consistent with the data carried by this block
	if !bytes.Equal(block.Data.Hash(), block.Header.DataHash) {
		return fmt.Errorf("Header.DataHash is different from Hash(block.Data) for block with id [%d] on channel [%s]", block.Header.Number, chainID)
	}

	//// - Get Policy for block validation
	//
	//// Get the policy manager for channelID
	//cpm, _ := s.channelPolicyManagerGetter.Manager(channelID)
	//if cpm == nil {
	//	return fmt.Errorf("Could not acquire policy manager for channel %s", channelID)
	//}
	//
	//// Get block validation policy
	//policy, _ := cpm.GetPolicy(policies.BlockValidation)

	// - Prepare SignedData
	signatureSet := []*pcommon.SignedData{}
	for _, metadataSignature := range metadata.Signatures {
		shdr, err := utils.GetSignatureHeader(metadataSignature.SignatureHeader)
		if err != nil {
			return fmt.Errorf("Failed unmarshalling signature header for block with id [%d] on channel [%s]: [%s]", block.Header.Number, chainID, err)
		}
		signatureSet = append(
			signatureSet,
			&pcommon.SignedData{
				Identity:  shdr.Creator,
				Data:      util.ConcatenateBytes(metadata.Value, metadataSignature.SignatureHeader, block.Header.Bytes()),
				Signature: metadataSignature.Signature,
			},
		)
	}

	// - Evaluate policy
	//return policy.Evaluate(signatureSet)
	return nil
}
