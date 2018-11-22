package network

import (
	"fmt"
	"io"
	"time"
)

type WhoamiMessage struct {
	Version   uint32
	Timestamp time.Time
}

func NewWhoamiMessage() *WhoamiMessage {
	return &WhoamiMessage{}
}

func (msg *WhoamiMessage) Decode(reader io.Reader) error {
	version, err := ReadUint32(reader)
	if err != nil {
		return err
	}

	timestamp, err := ReadUint64(reader)
	if err != nil {
		return err
	}

	msg.Version = version
	msg.Timestamp = time.Unix(int64(timestamp), 0)

	return nil
}

func (msg *WhoamiMessage) Encode(writer io.Writer) error {
	err := WriteUint32(writer, msg.Version)
	if err != nil {
		return err
	}

	return WriteUint64(writer, uint64(msg.Timestamp.Unix()))
}

func (msg *WhoamiMessage) MsgType() string {
	return "whoami"
}

func (msg WhoamiMessage) String() string {
	return fmt.Sprintf("WhoamiMessage[version=%d, timestamp=%v]", msg.Version, msg.Timestamp)
}
