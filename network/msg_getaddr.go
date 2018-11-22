package network

import (
	"io"
)

type GetAddrMessage struct {
}

func NewGetAddrMessage() *GetAddrMessage {
	return &GetAddrMessage{}
}

func (msg *GetAddrMessage) Decode(reader io.Reader) error {
	return nil
}

func (msg *GetAddrMessage) Encode(writer io.Writer) error {
	return nil
}

func (msg *GetAddrMessage) MsgType() string {
	return "getaddr"
}

func (msg GetAddrMessage) String() string {
	return "GetAddrMessage[]"
}
