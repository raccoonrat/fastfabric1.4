package state

import (
	"github.com/hyperledger/fabric/fastfabric-extensions/parallel"
	"github.com/hyperledger/fabric/fastfabric-extensions/unmarshaled"
	"github.com/hyperledger/fabric/gossip/state"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/transientstore"
	common2 "github.com/hyperledger/fabric/gossip/common"
	"github.com/pkg/errors"
)
var logger = util.GetLogger(util.ServiceLogger, "")

type ledgerResources interface {
	ValidateBlock(block *unmarshaled.Block) error

	// StoreBlock deliver new block with underlined private data
	// returns missing transaction ids
	StoreBlock(block *unmarshaled.Block, data util.PvtDataCollections) error

	// StorePvtData used to persist private date into transient store
	StorePvtData(txid string, privData *transientstore.TxPvtReadWriteSetWithConfigInfo, blckHeight uint64) error

	// GetPvtDataAndBlockByNum get block by number and returns also all related private data
	// the order of private data in slice of PvtDataCollections doesn't imply the order of
	// transactions in the block related to these private data, to get the correct placement
	// need to read TxPvtData.SeqInBlock field
	GetPvtDataAndBlockByNum(seqNum uint64, peerAuthInfo common.SignedData) (*unmarshaled.Block, util.PvtDataCollections, error)

	// Get recent block sequence number
	LedgerHeight() (uint64, error)

	// Close ledgerResources
	Close()
}

type GossipStateProviderImpl struct {
	state.GossipStateProvider
	chainID string

	mediator *state.ServicesMediator
	ledgerResources
}

func NewGossipStateProvider(chainID string, services *state.ServicesMediator, ledger ledgerResources) state.GossipStateProvider {
	gsp := &GossipStateProviderImpl{state.NewGossipStateProvider(chainID, services, ledger), chainID, services,  ledger}
	go gsp.deliverPayloads()
	return gsp
}


func (s *GossipStateProviderImpl) deliverPayloads() {
	go s.commit()

	for pipelinedBlock := range parallel.ReadyForValidation{
		go s.validate(pipelinedBlock)
	}
	logger.Debug("State provider has been stopped, finishing to push new blocks.")
	return
}

func (s *GossipStateProviderImpl) commit() {
	for blockPromise := range parallel.ReadyToCommit{
		block, more := <- blockPromise
		if !more{
			continue
		}

		// Commit block with available private transactions
		if err := s.ledgerResources.StoreBlock(block, util.PvtDataCollections{}); err != nil {
			logger.Errorf("Got error while committing(%+v)", errors.WithStack(err))
			return
		}

		// Update ledger height
		s.mediator.UpdateLedgerHeight(block.Raw.Header.Number+1, common2.ChainID(s.chainID))
		logger.Debugf("[%s] Committed block [%d] with %d transaction(s)",
			s.chainID, block.Raw.Header.Number, len(block.Txs))
	}
}

func (s *GossipStateProviderImpl) validate(pipeline *parallel.Pipeline) {
	defer close(pipeline.Channel)
	if err := s.ledgerResources.ValidateBlock(pipeline.Block); err !=nil{
		logger.Errorf("Validation failed: %+v", err)
		return
	}
	pipeline.Channel <- pipeline.Block
}

