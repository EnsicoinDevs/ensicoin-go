package network

import (
	"bytes"
	"errors"
	"github.com/EnsicoinDevs/eccd/consensus"
	"io"
)

// Message represents a message following the Ensicoin protocol.
type Message interface {
	Decode(io.Reader) error
	Encode(io.Writer) error
	MsgType() string
}

type messageHeader struct {
	magic   uint32
	msgType string
	length  uint64
}

func readMessageHeader(reader io.Reader) (*messageHeader, error) {
	magic, err := ReadUint32(reader)
	if err != nil {
		return nil, err
	}

	var msgType [12]byte
	_, err = io.ReadFull(reader, msgType[:])
	if err != nil {
		return nil, err
	}

	length, err := ReadUint64(reader)
	if err != nil {
		return nil, err
	}

	return &messageHeader{
		magic:   magic,
		msgType: string(bytes.TrimRight(msgType[:], string(0))),
		length:  length,
	}, nil
}

func writeMessageHeader(writer io.Writer, header *messageHeader) error {
	err := WriteUint32(writer, header.magic)
	if err != nil {
		return err
	}

	var emptyType [12]byte
	copy(emptyType[:], header.msgType)

	_, err = writer.Write(emptyType[:])
	if err != nil {
		return err
	}

	return WriteUint64(writer, header.length)
}

func newMessageByType(msgType string) (Message, error) {
	var msg Message

	switch msgType {
	case "whoami":
		msg = NewWhoamiMessage()
	case "whoamiack":
		msg = NewWhoamiAckMessage()
	case "getaddr":
		msg = NewGetAddrMessage()
	case "addr":
		msg = NewAddrMessage()
	case "getblocks":
		msg = NewGetBlocksMessage()
	case "block":
		msg = NewBlockMessage()
	case "inv":
		msg = NewInvMessage()
	case "getdata":
		msg = NewGetDataMessage()
	case "notfound":
		msg = NewNotFoundMessage()
	case "tx":
		msg = NewTxMessage()
	case "getmempool":
		msg = NewGetMempoolMessage()
	case "2plus2is4":
		msg = NewTwoPlusTwoIsFourMessage()
	case "minus1thats3":
		msg = NewMinusOneThatsThreeMessage()
	default:
		return nil, errors.New("bad message type: <" + msgType + ">")
	}

	return msg, nil
}

// ReadMessage decodes a raw message from a io.Reader.
func ReadMessage(reader io.Reader) (Message, error) {
	header, err := readMessageHeader(reader)
	if err != nil {
		return nil, err
	}

	if header.magic != consensus.NETWORK_MAGIC_NUMBER {
		return nil, errors.New("bad magic number")
	}

	msg, err := newMessageByType(header.msgType)
	if err != nil {
		return nil, err
	}

	// TODO: maybe it's better to read the full payload here

	err = msg.Decode(reader)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// WriteMessage encodes a message and writes it in a io.Writer. It generates the header of the message.
func WriteMessage(writer io.Writer, msg Message) error {
	var buf bytes.Buffer
	err := msg.Encode(&buf)
	if err != nil {
		return err
	}

	payload := buf.Bytes()
	length := len(payload)

	header := messageHeader{
		magic:   consensus.NETWORK_MAGIC_NUMBER,
		msgType: msg.MsgType(),
		length:  uint64(length),
	}

	err = writeMessageHeader(writer, &header)
	if err != nil {
		return err
	}

	_, err = writer.Write(payload)
	if err != nil {
		return err
	}

	return nil
}
