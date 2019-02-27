package network

import (
	"io"
)

type TwoPlusTwoIsFourMessage struct {
}

func NewTwoPlusTwoIsFourMessage() *TwoPlusTwoIsFourMessage {
	return &TwoPlusTwoIsFourMessage{}
}

func (msg *TwoPlusTwoIsFourMessage) Decode(reader io.Reader) error {
	return nil
}

func (msg *TwoPlusTwoIsFourMessage) Encode(writer io.Writer) error {
	return nil
}

func (msg *TwoPlusTwoIsFourMessage) MsgType() string {
	return "2plus2is4"
}

func (msg TwoPlusTwoIsFourMessage) String() string {
	return "2Plus2Is4[]"
}
