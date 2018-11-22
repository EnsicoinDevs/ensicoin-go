package network

import (
	"io"
	"net"
	"time"
)

type Address struct {
	Timestamp time.Time
	IP        net.IP
	Port      uint16
}

func readAddress(reader io.Reader) (*Address, error) {
	timestamp, err := ReadUint64(reader)
	if err != nil {
		return nil, err
	}

	ip := make([]byte, 16)
	if _, err = io.ReadFull(reader, ip); err != nil {
		return nil, err
	}

	port, err := ReadUint16(reader)
	if err != nil {
		return nil, err
	}

	return &Address{
		Timestamp: time.Unix(int64(timestamp), 0),
		IP:        net.IP(ip),
		Port:      port,
	}, nil
}

type AddrMessage struct {
	Addresses []*Address
}

func NewAddrMessage() *AddrMessage {
	return &AddrMessage{}
}

func (msg *AddrMessage) Decode(reader io.Reader) error {
	count, err := ReadVarUint(reader)
	if err != nil {
		return err
	}

	for i := uint64(0); i < count; i++ {
		address, err := readAddress(reader)
		if err != nil {
			return err
		}

		msg.Addresses = append(msg.Addresses, address)
	}

	return nil
}

func (msg *AddrMessage) Encode(writer io.Writer) error {
	return nil
}

func (msg *AddrMessage) MsgType() string {
	return "addr"

}

func (msg AddrMessage) String() string {
	return "AddrMessage[]"
}
