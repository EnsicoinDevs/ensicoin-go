package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

type BlockHeader struct {
	Version          int       `json:"version"`
	Flags            []string  `json:"flags"`
	HashPrevBlock    string    `json:"hashPrevBlock"`
	HashTransactions string    `json:"hashTransactions"`
	Timestamp        time.Time `json:"timestamp"`
	Nonce            int       `json:"nonce"`
}

type Block struct {
	Hash         string      `json:"hash"`
	Header       BlockHeader `json:"header"`
	Transactions []*Tx       `json:"transactions"`
}

func (block *Block) Validate() bool {
	return true
}

func (block *Block) ComputeHash() {
	h := sha256.New()

	h.Write([]byte(strconv.Itoa(block.Header.Version)))
	h.Write([]byte(strings.Join(block.Header.Flags, "")))
	h.Write([]byte(block.Header.HashPrevBlock))
	h.Write([]byte(block.Header.HashTransactions))
	h.Write([]byte(strconv.FormatInt(block.Header.Timestamp.Unix(), 10)))
	h.Write([]byte(strconv.Itoa(block.Header.Nonce)))
	v := h.Sum(nil)

	h.Reset()

	h.Write(v)

	block.Hash = hex.EncodeToString(h.Sum(nil))
}
