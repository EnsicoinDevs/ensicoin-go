package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"strconv"
	"strings"
	"time"
)

type BlockHeader struct {
	Version       int       `json:"version"`
	Flags         []string  `json:"flags"`
	HashPrevBlock string    `json:"hashPrevBlock"`
	HashTxs       string    `json:"hashTransactions"`
	Timestamp     time.Time `json:"timestamp"`
	Nonce         int       `json:"nonce"`
}

type Block struct {
	Hash   string      `json:"hash"`
	Header BlockHeader `json:"header"`
	Txs    []*Tx       `json:"transactions"`
}

func (block *Block) Validate() bool {
	return true
}

func (block *Block) ComputeHash() {
	h := sha256.New()

	h.Write([]byte(strconv.Itoa(block.Header.Version)))
	h.Write([]byte(strings.Join(block.Header.Flags, "")))
	h.Write([]byte(block.Header.HashPrevBlock))
	h.Write([]byte(block.Header.HashTxs))
	h.Write([]byte(strconv.FormatInt(block.Header.Timestamp.Unix(), 10)))
	h.Write([]byte(strconv.Itoa(block.Header.Nonce)))
	v := h.Sum(nil)

	h.Reset()

	h.Write(v)

	block.Hash = hex.EncodeToString(h.Sum(nil))
}

func (block *Block) ToBlockMessage() *network.BlockMessage {
	msg := network.BlockMessage{
		Header: network.BlockHeader{
			Version:       block.Header.Version,
			Flags:         block.Header.Flags,
			HashPrevBlock: block.Header.HashPrevBlock,
			HashTxs:       block.Header.HashTxs,
			Timestamp:     block.Header.Timestamp,
			Nonce:         block.Header.Nonce,
		},
	}

	for _, tx := range block.Txs {
		msg.Txs = append(msg.Txs, *tx.ToTxMessage())
	}

	return &msg
}
