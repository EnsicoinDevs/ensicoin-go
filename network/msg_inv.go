package network

import (
	"fmt"
	"github.com/EnsicoinDevs/ensicoincoin/utils"
	"io"
)

type InvVectType uint32

const (
	INV_VECT_TX    InvVectType = 0
	INV_VECT_BLOCK InvVectType = 1
)

type InvVect struct {
	InvType InvVectType
	Hash    *utils.Hash
}

func readInvVect(reader io.Reader) (*InvVect, error) {
	invType, err := ReadUint32(reader)
	if err != nil {
		return nil, err
	}

	hash, err := ReadHash(reader)
	if err != nil {
		return nil, err
	}

	return &InvVect{
		InvType: InvVectType(invType),
		Hash:    hash,
	}, nil
}

func writeInvVect(writer io.Writer, invVect *InvVect) error {
	err := WriteUint32(writer, uint32(invVect.InvType))
	if err != nil {
		return err
	}

	return WriteHash(writer, invVect.Hash)
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
	err := WriteVarUint(writer, uint64(len(msg.Inventory)))
	if err != nil {
		return err
	}

	for _, invVect := range msg.Inventory {
		err = writeInvVect(writer, invVect)
		if err != nil {
			return err
		}
	}

	return nil
}

func (msg *InvMessage) MsgType() string {
	return "inv"
}

func (msg InvMessage) String() string {
	return fmt.Sprintf("InvMessage[Inventory: %v]", msg.Inventory)
}
