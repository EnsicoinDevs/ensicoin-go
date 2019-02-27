package network

import (
	"io"
)

type MinusOneThatsThreeMessage struct {
}

func NewMinusOneThatsThreeMessage() *MinusOneThatsThreeMessage {
	return &MinusOneThatsThreeMessage{}
}

func (msg *MinusOneThatsThreeMessage) Decode(reader io.Reader) error {
	return nil
}

func (msg *MinusOneThatsThreeMessage) Encode(writer io.Writer) error {
	return nil
}

func (msg *MinusOneThatsThreeMessage) MsgType() string {
	return "minus1thats3"
}

func (msg MinusOneThatsThreeMessage) String() string {
	return "Minus1Thats3[]"
}
