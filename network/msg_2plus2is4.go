package network

import (
	"io"
)

// TwoPlusTwoIsFourMessage represents the message 2plus2is4.
type TwoPlusTwoIsFourMessage struct {
}

// NewTwoPlusTwoIsFourMessage returns a TwoPlusTwoIsFourMessage instance.
func NewTwoPlusTwoIsFourMessage() *TwoPlusTwoIsFourMessage {
	return &TwoPlusTwoIsFourMessage{}
}

// Decode implements the Message interface.
func (msg *TwoPlusTwoIsFourMessage) Decode(reader io.Reader) error {
	return nil
}

// Encode implements the Message interface.
func (msg *TwoPlusTwoIsFourMessage) Encode(writer io.Writer) error {
	return nil
}

// MsgType implements the Message interface.
func (msg *TwoPlusTwoIsFourMessage) MsgType() string {
	return "2plus2is4"
}

func (msg TwoPlusTwoIsFourMessage) String() string {
	return "2Plus2Is4[]"
}
