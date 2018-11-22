package network

import (
	"io"
)

type InvVectType uint32

const (
	INV_VECT_TX    = 0
	INV_VECT_BLOCK = 1
)

type InvVect struct {
	InvType InvVectType
	Hash    string
}

func readInvVect(reader io.Reader) (*InvVect, error) {
	invType, err := ReadUint32(reader)
	if err != nil {
		return nil, err
	}

	hash, err := ReadString(reader, 32)
	if err != nil {
		return nil, err
	}

	return &InvVect{
		InvType: InvVectType(invType),
		Hash:    hash,
	}, nil
}

type InvMessage struct {
	Inventory []*InvVect
}

func NewInvMessage() *InvMessage {
	return &InvMessage{}
}

func (msg *InvMessage) Decode(reader io.Reader) error {
	count, err := ReadVarUint(reader)
	if err != nil {
		return err
	}

	for i := uint64(0); i < count; i++ {
		invVect, err := readInvVect(reader)
		if err != nil {
			return err
		}

		msg.Inventory = append(msg.Inventory, invVect)
	}

	return nil
}

func (msg *InvMessage) Encode(writer io.Writer) error {
	return nil
}

func (msg *InvMessage) MsgType() string {
	return "inv"
}

func (msg InvMessage) String() string {
	return "InvMessage[]"
}
