/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils_test

import (
	"testing"

	"github.com/golang/protobuf/proto"
	configtxtest "github.com/hyperledger/fabric/common/configtx/test"
	"github.com/hyperledger/fabric/protos/common"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

var testChainID = "myuniquetestchainid"

func TestGetBlockFromBlockBytes(t *testing.T) {
	testChainID := "myuniquetestchainid"
	gb, err := configtxtest.MakeGenesisBlock(testChainID)
	assert.NoError(t, err, "Failed to create test configuration block")
	blockBytes, err := utils.Marshal(gb)
	assert.NoError(t, err, "Failed to marshal block")
	_, err = utils.GetBlockFromBlockBytes(blockBytes)
	assert.NoError(t, err, "to get block from block bytes")

	// bad block bytes
	_, err = utils.GetBlockFromBlockBytes([]byte("bad block"))
	assert.Error(t, err, "Expected error for malformed block bytes")
}

func TestGetMetadataFromNewBlock(t *testing.T) {
	block := common.NewBlock(0, nil)
	md, err := utils.GetMetadataFromBlock(block, cb.BlockMetadataIndex_ORDERER)
	assert.NoError(t, err, "Unexpected error extracting metadata from new block")
	assert.Nil(t, md.Value, "Expected metadata field value to be nil")
	assert.Equal(t, 0, len(md.Value), "Expected length of metadata field value to be 0")
	md, err = utils.GetMetadataFromBlock(block, cb.BlockMetadataIndex_ORDERER)
	if err != nil {
		panic(err)
	}
	assert.NotNil(t, md, "Expected to get metadata from block")

	// malformed metadata
	block.Metadata.Metadata[cb.BlockMetadataIndex_ORDERER] = []byte("bad metadata")
	_, err = utils.GetMetadataFromBlock(block, cb.BlockMetadataIndex_ORDERER)
	assert.Error(t, err, "Expected error with malformed metadata")
	assert.Panics(t, func() {
		_, err = utils.GetMetadataFromBlock(block, cb.BlockMetadataIndex_ORDERER)
		if err != nil {
			panic(err)
		}
	}, "Expected panic with malformed metadata")
}

func TestInitBlockMeta(t *testing.T) {
	// block with no metadata
	block := &cb.Block{}
	utils.InitBlockMetadata(block)
	// should have 3 entries
	assert.Equal(t, 3, len(block.Metadata.Metadata), "Expected block to have 3 metadata entries")

	// block with a single entry
	block = &cb.Block{
		Metadata: &cb.BlockMetadata{},
	}
	block.Metadata.Metadata = append(block.Metadata.Metadata, []byte{})
	utils.InitBlockMetadata(block)
	// should have 3 entries
	assert.Equal(t, 3, len(block.Metadata.Metadata), "Expected block to have 3 metadata entries")
}

func TestCopyBlockMetadata(t *testing.T) {
	srcBlock := common.NewBlock(0, nil)
	dstBlock := &cb.Block{}

	metadata, _ := proto.Marshal(&cb.Metadata{
		Value: []byte("orderer metadata"),
	})
	srcBlock.Metadata.Metadata[cb.BlockMetadataIndex_ORDERER] = metadata
	utils.CopyBlockMetadata(srcBlock, dstBlock)

	// check that the copy worked
	assert.Equal(t, len(srcBlock.Metadata.Metadata), len(dstBlock.Metadata.Metadata),
		"Expected target block to have same number of metadata entries after copy")
	assert.Equal(t, metadata, dstBlock.Metadata.Metadata[cb.BlockMetadataIndex_ORDERER],
		"Unexpected metadata from target block")
}

func TestGetLastConfigIndexFromBlock(t *testing.T) {
	block := common.NewBlock(0, nil)
	index := uint64(2)
	lc, _ := proto.Marshal(&cb.LastConfig{
		Index: index,
	})
	metadata, _ := proto.Marshal(&cb.Metadata{
		Value: lc,
	})
	block.Metadata.Metadata[cb.BlockMetadataIndex_LAST_CONFIG] = metadata
	result, err := utils.GetLastConfigIndexFromBlock(block)
	assert.NoError(t, err, "Unexpected error returning last config index")
	assert.Equal(t, index, result, "Unexpected last config index returned from block")
	result = utils.GetLastConfigIndexFromBlockOrPanic(block)
	assert.Equal(t, index, result, "Unexpected last config index returned from block")

	// malformed metadata
	block.Metadata.Metadata[cb.BlockMetadataIndex_LAST_CONFIG] = []byte("bad metadata")
	_, err = utils.GetLastConfigIndexFromBlock(block)
	assert.Error(t, err, "Expected error with malformed metadata")

	// malformed last config
	metadata, _ = proto.Marshal(&cb.Metadata{
		Value: []byte("bad last config"),
	})
	block.Metadata.Metadata[cb.BlockMetadataIndex_LAST_CONFIG] = metadata
	_, err = utils.GetLastConfigIndexFromBlock(block)
	assert.Error(t, err, "Expected error with malformed last config metadata")
	assert.Panics(t, func() {
		_ = utils.GetLastConfigIndexFromBlockOrPanic(block)
	}, "Expected panic with malformed last config metadata")
}
