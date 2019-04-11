/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blocksprovider

import (
	"github.com/hyperledger/fabric/core/deliverservice/blocksprovider"
	"github.com/hyperledger/fabric/fastfabric-extensions/cached"
	"github.com/hyperledger/fabric/fastfabric-extensions/config"
	"github.com/hyperledger/fabric/fastfabric-extensions/parallel"
	"math"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/gossip/api"
	gossipcommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/hyperledger/fabric/protos/common"
	gossip_proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/orderer"
)

type streamClient interface {
	blocksprovider.BlocksDeliverer

	// UpdateEndpoint update ordering service endpoints
	UpdateEndpoints(endpoints []string)

	// GetEndpoints
	GetEndpoints() []string

	// Close closes the stream and its underlying connection
	Close()

	// Disconnect disconnects from the remote node and disable reconnect to current endpoint for predefined period of time
	Disconnect(disableEndpoint bool)
}

// blocksProviderImpl the actual implementation for BlocksProvider interface
type blocksProviderImpl struct {
	chainID string

	client streamClient

	gossip blocksprovider.GossipServiceAdapter

	mcs api.MessageCryptoService

	done int32

	wrongStatusThreshold int
}

const wrongStatusThreshold = 10

var maxRetryDelay = time.Second * 10
var logger = flogging.MustGetLogger("blocksProvider")

// NewBlocksProvider constructor function to create blocks deliverer instance
func NewBlocksProvider(
	chainID string,
	client streamClient,
	gossip blocksprovider.GossipServiceAdapter,
	mcs api.MessageCryptoService) blocksprovider.BlocksProvider {
	return &blocksProviderImpl{
		chainID:              chainID,
		client:               client,
		gossip:               gossip,
		mcs:                  mcs,
		wrongStatusThreshold: wrongStatusThreshold,
	}
}

// DeliverBlocks used to pull out blocks from the ordering service to
// distributed them across peers
func (b *blocksProviderImpl) DeliverBlocks() {
	errorStatusCounter := 0
	statusCounter := 0
	defer b.client.Close()
	defer close(parallel.ReadyForValidation)

	messages := make(chan *orderer.DeliverResponse, config.BlockPipelineWidth)
	go b.receive(messages)
	for msg := range messages {
		switch t := msg.Type.(type) {
		case *orderer.DeliverResponse_Status:
			if t.Status == common.Status_SUCCESS {
				logger.Warningf("[%s] ERROR! Received success for a seek that should never complete", b.chainID)
				return
			}
			if t.Status == common.Status_BAD_REQUEST || t.Status == common.Status_FORBIDDEN {
				logger.Errorf("[%s] Got error %v", b.chainID, t)
				errorStatusCounter++
				if errorStatusCounter > b.wrongStatusThreshold {
					logger.Criticalf("[%s] Wrong statuses threshold passed, stopping block provider", b.chainID)
					return
				}
			} else {
				errorStatusCounter = 0
				logger.Warningf("[%s] Got error %v", b.chainID, t)
			}
			maxDelay := float64(maxRetryDelay)
			currDelay := float64(time.Duration(math.Pow(2, float64(statusCounter))) * 100 * time.Millisecond)
			time.Sleep(time.Duration(math.Min(maxDelay, currDelay)))
			if currDelay < maxDelay {
				statusCounter++
			}
			if t.Status == common.Status_BAD_REQUEST {
				b.client.Disconnect(false)
			} else {
				b.client.Disconnect(true)
			}
			continue
		case *orderer.DeliverResponse_Block:
			errorStatusCounter = 0
			statusCounter = 0

			commitPromise := make(chan *cached.Block, 1)
			go b.processBlock(t.Block, commitPromise)
			parallel.ReadyToCommit <- commitPromise


		default:
			logger.Warningf("[%s] Received unknown: %v", b.chainID, t)
			return
		}
	}
}

func (b *blocksProviderImpl) receive(messages chan *orderer.DeliverResponse) {
	defer close(messages)
	for !b.isDone() {
		msg, err := b.client.Recv()
		if err != nil {
			logger.Warningf("[%s] Receive error: %s", b.chainID, err.Error())
			return
		}
		messages <- msg
	}
}

func (b *blocksProviderImpl) processBlock(block *common.Block, commitPromise chan *cached.Block) {
	done := b.verify(block)
	success := false
	//if done is closed without successful verification, then the loop will not be executed
	for verifiedBlock := range done{
		parallel.ReadyForValidation <- &parallel.Pipeline{commitPromise, verifiedBlock}
		success = true
		b.gossipBlock(block)
	}

	if !success{
		close(commitPromise)
	}
}

func (b *blocksProviderImpl) verify(block *common.Block) (chan *cached.Block){
	done := make(chan *cached.Block,1)
	defer close(done)
	cblock := cached.GetBlock(block)
	if err := b.mcs.VerifyBlock(gossipcommon.ChainID(b.chainID),cblock.Header.Number, cblock); err != nil {
		logger.Errorf("[%s] Error verifying block with sequnce number %d, due to %s", b.chainID, cblock.Header.Number, err)
		return done
	}

	done <- cblock
	return done
}

func (b *blocksProviderImpl) gossipBlock(block *common.Block) {
	marshaledBlock, err := proto.Marshal(block)
	blockNum := block.Header.Number
	if err != nil {
		logger.Errorf("[%s] Error serializing block with sequence number %d, due to %s", b.chainID,blockNum , err)
		return
	}

	numberOfPeers := len(b.gossip.PeersOfChannel(gossipcommon.ChainID(b.chainID)))
	// Create payload with a block received
	payload := createPayload(blockNum, marshaledBlock)
	// Use payload to create gossip message
	gossipMsg := createGossipMsg(b.chainID, payload)

	// Gossip messages with other nodes
	logger.Debugf("[%s] Gossiping block [%d], peers number [%d]", b.chainID, blockNum, numberOfPeers)
	if !b.isDone() {
		b.gossip.Gossip(gossipMsg)
	}
}

// Stop stops blocks delivery provider
func (b *blocksProviderImpl) Stop() {
	atomic.StoreInt32(&b.done, 1)
	b.client.Close()
}

// UpdateOrderingEndpoints update endpoints of ordering service
func (b *blocksProviderImpl) UpdateOrderingEndpoints(endpoints []string) {
	if !b.isEndpointsUpdated(endpoints) {
		// No new endpoints for ordering service were provided
		return
	}
	// We have got new set of endpoints, updating client
	logger.Debug("Updating endpoint, to %s", endpoints)
	b.client.UpdateEndpoints(endpoints)
	logger.Debug("Disconnecting so endpoints update will take effect")
	// We need to disconnect the client to make it reconnect back
	// to newly updated endpoints
	b.client.Disconnect(false)
}
func (b *blocksProviderImpl) isEndpointsUpdated(endpoints []string) bool {
	if len(endpoints) != len(b.client.GetEndpoints()) {
		return true
	}
	// Check that endpoints was actually updated
	for _, endpoint := range endpoints {
		if !util.Contains(endpoint, b.client.GetEndpoints()) {
			// Found new endpoint
			return true
		}
	}
	// Nothing has changed
	return false
}

// Check whenever provider is stopped
func (b *blocksProviderImpl) isDone() bool {
	return atomic.LoadInt32(&b.done) == 1
}

func createGossipMsg(chainID string, payload *gossip_proto.Payload) *gossip_proto.GossipMessage {
	gossipMsg := &gossip_proto.GossipMessage{
		Nonce:   0,
		Tag:     gossip_proto.GossipMessage_CHAN_AND_ORG,
		Channel: []byte(chainID),
		Content: &gossip_proto.GossipMessage_DataMsg{
			DataMsg: &gossip_proto.DataMessage{
				Payload: payload,
			},
		},
	}
	return gossipMsg
}

func createPayload(seqNum uint64, marshaledBlock []byte) *gossip_proto.Payload {
	return &gossip_proto.Payload{
		Data:   marshaledBlock,
		SeqNum: seqNum,
	}
}
