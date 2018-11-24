package network

import (
	"github.com/EnsicoinDevs/ensicoincoin/utils"
	"io"
)

type GetBlocksMessage struct {
	BlockLocator []*utils.Hash
	HashStop     *utils.Hash
}

func NewGetBlocksMessage() *GetBlocksMessage {
	return &GetBlocksMessage{}
}

func (msg *GetBlocksMessage) Decode(reader io.Reader) error {
	count, err := ReadVarUint(reader)
	if err != nil {
		return err
	}

	for i := uint64(0); i < count; i++ {
		hash, err := ReadHash(reader)
		if err != nil {
			return err
		}

		msg.BlockLocator = append(msg.BlockLocator, hash)
	}

	msg.HashStop, err = ReadHash(reader)

	return err
}

func (msg *GetBlocksMessage) Encode(writer io.Writer) error {
	err := WriteVarUint(writer, uint64(len(msg.BlockLocator)))
	if err != nil {
		return err
	}

	for _, hash := range msg.BlockLocator {
		err = WriteHash(writer, hash)
		if err != nil {
			return err
		}
	}

	return WriteHash(writer, msg.HashStop)
}

func (msg *GetBlocksMessage) MsgType() string {
	return "getblocks"
}

func (msg GetBlocksMessage) String() string {
	return "GetBlocksMessage[]"
}
