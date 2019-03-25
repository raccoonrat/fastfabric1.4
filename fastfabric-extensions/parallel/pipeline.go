package parallel

import (
	"github.com/hyperledger/fabric/fastfabric-extensions/config"
	"github.com/hyperledger/fabric/fastfabric-extensions/unmarshaled"
)

var ReadyToCommit = make(chan chan *unmarshaled.Block, config.BlockPipelineWidth)
var ReadyForValidation = make(chan *Pipeline, config.BlockPipelineWidth)


type Pipeline struct {
	Channel chan *unmarshaled.Block
	Block *unmarshaled.Block
}