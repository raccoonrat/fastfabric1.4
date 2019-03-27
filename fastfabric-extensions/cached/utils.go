package cached

import (
	"bytes"
	"fmt"
)

func (b *Block) VerifyDataHash() error{
	if b.Header == nil {
		return fmt.Errorf("block [%d] header is nil", b.Header)
	}

	if !bytes.Equal(b.Data.Hash(), b.Header.DataHash ) {
		return fmt.Errorf("Header.DataHash is different from Hash(block.Data) for block with id [%d]", b.Header.Number)
	}
	return nil
}

func (b *Block) GetChannelId()(string, error) {
	env, err := b.unmarshalSpecificData(0)
	if err != nil {
		return "", err
	}
	return env.GetChannelId()
}

func (env *Envelope) GetChannelId() (string, error) {
	pl,err := env.UnmarshalPayload()
	if err != nil{
		return "", err
	}
	hdr, err := pl.UnmarshalChannelHeader()
	if err != nil{
		return "", err
	}
	return hdr.ChannelId, nil
}


