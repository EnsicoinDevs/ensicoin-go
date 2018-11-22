package network

import (
	"io"
)

type WhoamiAckMessage struct {
}

func NewWhoamiAckMessage() *WhoamiAckMessage {
	return &WhoamiAckMessage{}
}

func (msg *WhoamiAckMessage) Decode(reader io.Reader) error {
	return nil
}

func (msg *WhoamiAckMessage) Encode(writer io.Writer) error {
	return nil
}

func (msg *WhoamiAckMessage) MsgType() string {
	return "whoamiack"
}

func (msg WhoamiAckMessage) String() string {
	return "WhoamiAckMessage[]"
}
