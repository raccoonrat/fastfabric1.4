package pipeline

import "github.com/hyperledger/fabric/fastfabric-extensions/config"

var BlockNums = make(chan uint64, config.BlockPipelineWidth)
var VerifiedBlockNums = make(chan uint64, config.BlockPipelineWidth)
var ValidatedBlockNums = make(chan uint64, config.BlockPipelineWidth)