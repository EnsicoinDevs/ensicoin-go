package network

import (
	"io"
	"time"
)

type BlockMessage struct {
	Version        uint32
	Flags          []string
	HashPrevBlock  string
	HashMerkleRoot string
	Timestamp      time.Time
	Bits           uint32
	Nonce          uint32
	Txs            []*TxMessage
}

func NewBlockMessage() *BlockMessage {
	return &BlockMessage{}
}

func (msg *BlockMessage) Decode(reader io.Reader) error {
	return nil
}

func (msg *BlockMessage) Encode(writer io.Writer) error {
	return nil
}

func (msg *BlockMessage) MsgType() string {
	return "block"
}

func (msg BlockMessage) String() string {
	return "MsgBlock[]"
}
