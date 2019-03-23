package experiment

type ExperimentParams struct {
	BlockParams  []BlockParams
	ThreadParams []ThreadParams
	GrpcAddress  string
	ChannelId    string
}
type BlockParams struct {
	BlockSize   int
	Repetitions int
}

type ThreadParams struct {
	BlockPipeline int
	TxPipeline    int
}

var Current ExperimentParams
var IsEndorser = false
