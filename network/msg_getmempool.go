package network

import (
	"io"
)

type GetMempoolMessage struct {
}

func NewGetMempoolMessage() *GetMempoolMessage {
	return &GetMempoolMessage{}
}

func (msg *GetMempoolMessage) Decode(reader io.Reader) error {
	return nil
}

func (msg *GetMempoolMessage) Encode(writer io.Writer) error {
	return nil
}

func (msg *GetMempoolMessage) MsgType() string {
	return "getmempool"
}

func (msg GetMempoolMessage) String() string {
	return "GetMempoolMessage[]"
}
