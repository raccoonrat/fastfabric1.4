package parallel

import (
	"github.com/hyperledger/fabric/fastfabric-extensions/config"
	"github.com/hyperledger/fabric/fastfabric-extensions/cached"
)

var ReadyToCommit = make(chan chan *cached.Block, config.BlockPipelineWidth)
var ReadyForValidation = make(chan *Pipeline, config.BlockPipelineWidth)


type Pipeline struct {
	Channel chan *cached.Block
	Block *cached.Block
}