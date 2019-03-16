package commit

import "github.com/fabric_extension"

var sequence chan uint64

func GetSequence()chan uint64{
	if sequence == nil{
		sequence = make(chan uint64, fabric_extension.PipelineWidth)
	}
	return sequence
}

var verified chan uint64

func GetVerified()chan uint64{
	if verified == nil{
		verified = make(chan uint64, fabric_extension.PipelineWidth)
	}
	return verified
}

var validated chan uint64

func GetValidated()chan uint64{
	if validated == nil{
		validated = make(chan uint64, fabric_extension.PipelineWidth)
	}
	return validated
}