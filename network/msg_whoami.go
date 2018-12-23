package network

import (
	"fmt"
	"io"
)

type WhoamiMessage struct {
	Version uint32
	From    *Address
}

func NewWhoamiMessage() *WhoamiMessage {
	return &WhoamiMessage{}
}

func (msg *WhoamiMessage) Decode(reader io.Reader) error {
	version, err := ReadUint32(reader)
	if err != nil {
		return err
	}

	address, err := ReadAddress(reader)
	if err != nil {
		return err
	}

	msg.Version = version
	msg.From = address

	return nil
}

func (msg *WhoamiMessage) Encode(writer io.Writer) error {
	err := WriteUint32(writer, msg.Version)
	if err != nil {
		return err
	}

	return WriteAddress(writer, msg.From)
}

func (msg *WhoamiMessage) MsgType() string {
	return "whoami"
}

func (msg WhoamiMessage) String() string {
	return fmt.Sprintf("WhoamiMessage[version=%d, from=%v]", msg.Version, msg.From)
}
