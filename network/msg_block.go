package network

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/EnsicoinDevs/eccd/utils"
	"io"
	"math/big"
	"time"
)

type BlockHeader struct {
	Version        uint32
	Flags          []string
	HashPrevBlock  *utils.Hash
	HashMerkleRoot *utils.Hash
	Timestamp      time.Time
	Height         uint32
	Target         *big.Int
	Nonce          uint64
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

	hashPrevBlock, err := ReadHash(reader)
	if err != nil {
		return nil, err
	}

	hashMerkleRoot, err := ReadHash(reader)
	if err != nil {
		return nil, err
	}

	timestamp, err := ReadUint64(reader)
	if err != nil {
		return nil, err
	}

	height, err := ReadUint32(reader)
	if err != nil {
		return nil, err
	}

	targetHash, err := ReadHash(reader)
	if err != nil {
		return nil, err
	}

	nonce, err := ReadUint64(reader)
	if err != nil {
		return nil, err
	}

	return &BlockHeader{
		Version:        version,
		Flags:          flags,
		HashPrevBlock:  hashPrevBlock,
		HashMerkleRoot: hashMerkleRoot,
		Timestamp:      time.Unix(int64(timestamp), 0),
		Height:         height,
		Target:         targetHash.Big(),
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

	err = WriteHash(writer, header.HashPrevBlock)
	if err != nil {
		return err
	}

	err = WriteHash(writer, header.HashMerkleRoot)
	if err != nil {
		return err
	}

	err = WriteUint64(writer, uint64(header.Timestamp.Unix()))
	if err != nil {
		return err
	}

	err = WriteUint32(writer, header.Height)
	if err != nil {
		return err
	}

	err = WriteHash(writer, utils.BigToHash(header.Target))
	if err != nil {
		return err
	}

	err = WriteUint64(writer, header.Nonce)
	if err != nil {
		return err
	}

	return nil
}

func (header *BlockHeader) Hash() *utils.Hash {
	buf := bytes.NewBuffer(nil)
	_ = writeBlockHeader(buf, header)

	hash := utils.Hash(sha256.Sum256(buf.Bytes()))

	hash = sha256.Sum256(hash.Bytes())

	return &hash
}

func (header *BlockHeader) HashString() string {
	return hex.EncodeToString(header.Hash().Bytes())
}

type BlockMessage struct {
	Header *BlockHeader
	Txs    []*TxMessage
}

func NewBlockMessage() *BlockMessage {
	return &BlockMessage{}
}

func (msg *BlockMessage) Decode(reader io.Reader) error {
	header, err := readBlockHeader(reader)
	if err != nil {
		return err
	}

	msg.Header = header

	count, err := ReadVarUint(reader)
	if err != nil {
		return err
	}

	for i := uint64(0); i < count; i++ {
		tx := NewTxMessage()

		err = tx.Decode(reader)
		if err != nil {
			return err
		}

		msg.Txs = append(msg.Txs, tx)
	}

	return nil
}

func (msg *BlockMessage) Encode(writer io.Writer) error {
	err := writeBlockHeader(writer, msg.Header)
	if err != nil {
		return err
	}

	err = WriteVarUint(writer, uint64(len(msg.Txs)))
	if err != nil {
		return err
	}

	for _, tx := range msg.Txs {
		err = tx.Encode(writer)
		if err != nil {
			return err
		}
	}

	return nil
}

func (msg *BlockMessage) MsgType() string {
	return "block"
}

func (msg BlockMessage) String() string {
	return fmt.Sprintf("MsgBlock[Version: %d, Flags: %v, HashPrevBlock: %x, HashMerkleRoot: %x, Timestamp: %v, Height: %d, Target: %x, Nonce: %d, Txs: %v]", msg.Header.Version, msg.Header.Flags, msg.Header.HashPrevBlock, msg.Header.HashMerkleRoot, msg.Header.Timestamp, msg.Header.Height, utils.BigToHash(msg.Header.Target), msg.Header.Nonce, msg.Txs)
}
