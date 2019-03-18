/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blocksprovider

import (
	"context"
	"github.com/fabric_extension"
	"github.com/fabric_extension/block_cache"
	"github.com/fabric_extension/commit"
	"golang.org/x/sync/semaphore"
	"math"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/gossip/api"
	gossipcommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/hyperledger/fabric/protos/common"
	gossip_proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/op/go-logging"
)

// LedgerInfo an adapter to provide the interface to query
// the ledger committer for current ledger height
type LedgerInfo interface {
	// LedgerHeight returns current local ledger height
	LedgerHeight() (uint64, error)
}

// GossipServiceAdapter serves to provide basic functionality
// required from gossip service by delivery service
type GossipServiceAdapter interface {
	// PeersOfChannel returns slice with members of specified channel
	PeersOfChannel(gossipcommon.ChainID) []discovery.NetworkMember

	// AddPayload adds payload to the local state sync buffer
	AddPayload(chainID string, payload *gossip_proto.Payload) error

	// Gossip the message across the peers
	Gossip(msg *gossip_proto.GossipMessage)
}

// BlocksProvider used to read blocks from the ordering service
// for specified chain it subscribed to
type BlocksProvider interface {
	// DeliverBlocks starts delivering and disseminating blocks
	DeliverBlocks()

	// UpdateClientEndpoints update endpoints
	UpdateOrderingEndpoints(endpoints []string)

	// Stop shutdowns blocks provider and stops delivering new blocks
	Stop()
}

// BlocksDeliverer defines interface which actually helps
// to abstract the AtomicBroadcast_DeliverClient with only
// required method for blocks provider.
// This also decouples the production implementation of the gRPC stream
// from the code in order for the code to be more modular and testable.
type BlocksDeliverer interface {
	// Recv retrieves a response from the ordering service
	Recv() (*orderer.DeliverResponse, error)

	// Send sends an envelope to the ordering service
	Send(*common.Envelope) error
}

type streamClient interface {
	BlocksDeliverer

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

	gossip GossipServiceAdapter

	mcs api.MessageCryptoService

	done int32

	wrongStatusThreshold int
}

const wrongStatusThreshold = 10

var maxRetryDelay = time.Second * 10

var logger *logging.Logger // package-level logger

func init() {
	logger = flogging.MustGetLogger("blocksProvider")
}

// NewBlocksProvider constructor function to create blocks deliverer instance
func NewBlocksProvider(chainID string, client streamClient, gossip GossipServiceAdapter, mcs api.MessageCryptoService) BlocksProvider {
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
	messages := make(chan *orderer.DeliverResponse,fabric_extension.PipelineWidth)
	go b.receive(messages)

	var weight int64
	if fabric_extension.PipelineWidth/2<1{
		weight = 1
	}else{
		weight = int64(fabric_extension.PipelineWidth/2)
	}
	weighted := semaphore.NewWeighted(weight)

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
			seqNum := t.Block.Header.Number
			blocks.Cache.Put(t.Block)
			commit.GetSequence() <-seqNum

			weighted.Acquire(context.Background(), 1)
			go func(w *semaphore.Weighted) {
				defer w.Release(1)

				if err := b.mcs.VerifyBlockByNo(gossipcommon.ChainID(b.chainID), seqNum); err != nil {
					logger.Errorf("[%s] Error verifying block with sequnce number %d, due to %s", b.chainID, seqNum, err)
					return
				}
				commit.GetVerified() <- seqNum

				go b.gossipBlock(t.Block)
			}(weighted)

		default:
			logger.Warningf("[%s] Received unknown: ", b.chainID, t)
			return
		}
	}
	close(commit.GetVerified())
	close(commit.GetSequence())
}
func (b *blocksProviderImpl) gossipBlock(block *common.Block) {
	numberOfPeers := len(b.gossip.PeersOfChannel(gossipcommon.ChainID(b.chainID)))
	marshaledBlock, err := proto.Marshal(block)
	seqNum := block.Header.Number
	if err != nil {
		logger.Errorf("[%s] Error serializing block with sequence number %d, due to %s", b.chainID, seqNum, err)
		return
	}
	// Create payload with a block received
	payload := createPayload(seqNum, marshaledBlock)
	// Use payload to create gossip message
	gossipMsg := createGossipMsg(b.chainID, payload)



	// Gossip messages with other nodes
	logger.Debugf("[%s] Gossiping block [%d], peers number [%d]", b.chainID, seqNum, numberOfPeers)
	if !b.isDone() {
		b.gossip.Gossip(gossipMsg)
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
