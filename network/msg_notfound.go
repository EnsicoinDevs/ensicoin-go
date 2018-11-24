package network

import (
	"io"
)

type NotFoundMessage struct {
	Inventory []*InvVect
}

func NewNotFoundMessage() *NotFoundMessage {
	return &NotFoundMessage{}
}

func (msg *NotFoundMessage) Decode(reader io.Reader) error {
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

func (msg *NotFoundMessage) Encode(writer io.Writer) error {
	err := WriteVarUint(writer, uint64(len(msg.Inventory)))
	if err != nil {
		return err
	}

	for _, invVect := range msg.Inventory {
		err = writeInvVect(writer, invVect)
		if err != nil {
			return nil
		}
	}

	return nil
}

func (msg *NotFoundMessage) MsgType() string {
	return "notfound"
}

func (msg NotFoundMessage) String() string {
	return "NotFoundMessage[]"
}
