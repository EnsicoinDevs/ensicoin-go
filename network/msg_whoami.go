package network

import (
	"fmt"
	"io"
)

type WhoamiMessage struct {
	Version  uint32
	From     *Address
	Services []string
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

	servicesCount, err := ReadVarUint(reader)
	if err != nil {
		return err
	}

	for i := uint64(0); i < servicesCount; i++ {
		service, err := ReadVarString(reader)
		if err != nil {
			return err
		}

		msg.Services = append(msg.Services, service)
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

	err = WriteAddress(writer, msg.From)
	if err != nil {
		return err
	}

	err = WriteVarUint(writer, uint64(len(msg.Services)))
	if err != nil {
		return err
	}

	for _, service := range msg.Services {
		err = WriteVarString(writer, service)
		if err != nil {
			return err
		}
	}

	return nil
}

func (msg *WhoamiMessage) MsgType() string {
	return "whoami"
}

func (msg WhoamiMessage) String() string {
	return fmt.Sprintf("WhoamiMessage[version=%d, from=%v]", msg.Version, msg.From)
}
