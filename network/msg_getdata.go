package network

import (
	"io"
)

type GetDataMessage struct {
	Inventory []*InvVect
}

func NewGetDataMessage() *GetDataMessage {
	return &GetDataMessage{}
}

func (msg *GetDataMessage) Decode(reader io.Reader) error {
	return nil
}

func (msg *GetDataMessage) Encode(writer io.Writer) error {
	return nil
}

func (msg *GetDataMessage) MsgType() string {
	return "getdata"
}

func (msg GetDataMessage) String() string {
	return "GetDataMessage[]"
}
