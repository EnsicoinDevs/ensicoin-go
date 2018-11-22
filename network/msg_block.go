package network

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"time"
)

type BlockHeader struct {
	Version        uint32
	Flags          []string
	HashPrevBlock  string
	HashMerkleRoot string
	Timestamp      time.Time
	Bits           uint32
	Nonce          uint32
}

func readBlockHeader(reader io.Reader) (*BlockHeader, error) {
	version, err := ReadUint32(reader)
	if err != nil {
		return nil, err
	}

	flagsCount, err := ReadVarUint(reader)
	if err != nil {
		return nil, err
	}

	var flags []string

	for i := uint64(0); i < flagsCount; i++ {
		flag, err := ReadVarString(reader)
		if err != nil {
			return nil, err
		}

		flags = append(flags, flag)
	}

	hashPrevBlock, err := ReadString(reader, 32)
	if err != nil {
		return nil, err
	}

	hashMerkleRoot, err := ReadString(reader, 32)
	if err != nil {
		return nil, err
	}

	timestamp, err := ReadUint64(reader)
	if err != nil {
		return nil, err
	}

	bits, err := ReadUint32(reader)
	if err != nil {
		return nil, err
	}

	nonce, err := ReadUint32(reader)
	if err != nil {
		return nil, err
	}

	return &BlockHeader{
		Version:        version,
		Flags:          flags,
		HashPrevBlock:  hashPrevBlock,
		HashMerkleRoot: hashMerkleRoot,
		Timestamp:      time.Unix(int64(timestamp), 0),
		Bits:           bits,
		Nonce:          nonce,
	}, nil
}

func writeBlockHeader(writer io.Writer, header *BlockHeader) error {
	err := WriteUint32(writer, header.Version)
	if err != nil {
		return err
	}

	err = WriteVarUint(writer, uint64(len(header.Flags)))
	if err != nil {
		return err
	}

	for _, flag := range header.Flags {
		err = WriteVarString(writer, flag)
		if err != nil {
			return err
		}
	}

	err = WriteString(writer, header.HashPrevBlock)
	if err != nil {
		return err
	}

	err = WriteString(writer, header.HashMerkleRoot)
	if err != nil {
		return err
	}

	err = WriteUint64(writer, uint64(header.Timestamp.Unix()))
	if err != nil {
		return err
	}

	err = WriteUint32(writer, header.Bits)
	if err != nil {
		return err
	}

	err = WriteUint32(writer, header.Nonce)
	if err != nil {
		return err
	}

	return nil
}

func (header *BlockHeader) Hash() []byte {
	buf := bytes.NewBuffer(make([]byte, 0, 232))
	_ = writeBlockHeader(buf, header)

	hash := sha256.Sum256(buf.Bytes())

	hash = sha256.Sum256(hash[:])

	return hash[:]
}

func (header *BlockHeader) HashString() string {
	return hex.EncodeToString(header.Hash()[:])
}

type BlockMessage struct {
	Txs []*TxMessage
}

func NewBlockMessage() *BlockMessage {
	return &BlockMessage{}
}

func (msg *BlockMessage) Decode(reader io.Reader) error {
	return nil
}

func (msg *BlockMessage) Encode(writer io.Writer) error {
	return nil
}

func (msg *BlockMessage) MsgType() string {
	return "block"
}

func (msg BlockMessage) String() string {
	return "MsgBlock[]"
}
