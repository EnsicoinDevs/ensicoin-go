package network

import (
	"fmt"
	"io"
)

type GetDataMessage struct {
	Inventory []*InvVect
}

func NewGetDataMessage() *GetDataMessage {
	return &GetDataMessage{}
}

func (msg *GetDataMessage) Decode(reader io.Reader) error {
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

func (msg *GetDataMessage) Encode(writer io.Writer) error {
	err := WriteVarUint(writer, uint64(len(msg.Inventory)))
	if err != nil {
		return nil
	}

	for _, invVect := range msg.Inventory {
		err = writeInvVect(writer, invVect)
		if err != nil {
			return err
		}
	}

	return nil
}

func (msg *GetDataMessage) MsgType() string {
	return "getdata"
}

func (msg GetDataMessage) String() string {
	return fmt.Sprintf("GetDataMessage[Inventory: %v]", msg.Inventory)
}
