package network

import (
	"io"
)

type Outpoint struct {
	Hash  string
	Index uint32
}

type TxIn struct {
	PreviousOutput *Outpoint
	Script         []byte
}

type TxOut struct {
	Value  uint64
	Script []byte
}

type TxMessage struct {
	Version uint32
	Flags   []string
	Inputs  []*TxIn
	Outputs []*TxOut
}

func NewTxMessage() *TxMessage {
	return &TxMessage{}
}

func (msg *TxMessage) Decode(reader io.Reader) error {
	return nil
}

func (msg *TxMessage) Encode(writer io.Writer) error {
	return nil
}

func (msg *TxMessage) MsgType() string {
	return "tx"
}

func (msg TxMessage) String() string {
	return "TxMessage[]"
}
