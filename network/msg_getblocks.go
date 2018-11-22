package network

import (
	"io"
)

type GetBlocksMessage struct {
	BlockLocator []string
	HashStop     string
}

func NewGetBlocksMessage() *GetBlocksMessage {
	return &GetBlocksMessage{}
}

func (msg *GetBlocksMessage) Decode(reader io.Reader) error {
	return nil
}

func (msg *GetBlocksMessage) Encode(writer io.Writer) error {
	return nil
}

func (msg *GetBlocksMessage) MsgType() string {
	return "getblocks"
}

func (msg GetBlocksMessage) String() string {
	return "GetBlocksMessage[]"
}
