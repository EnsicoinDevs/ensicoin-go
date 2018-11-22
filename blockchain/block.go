package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"time"
)

type BlockHeader struct {
	Version        uint32    `json:"version"`
	Flags          []string  `json:"flags"`
	HashPrevBlock  string    `json:"hashPrevBlock"`
	HashMerkleRoot string    `json:"hashTransactions"`
	Timestamp      time.Time `json:"timestamp"`
	Bits           uint32    `json:"bits"`
	Nonce          uint32    `json:"nonce"`
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

	// TODO: hash
	v := h.Sum(nil)

	h.Reset()

	h.Write(v)

	block.Hash = hex.EncodeToString(h.Sum(nil))
}

func (block *Block) ToBlockMessage() *network.BlockMessage {
	msg := network.BlockMessage{}

	for _, tx := range block.Txs {
		//	msg.Txs = append(msg.Txs, *tx.ToTxMessage())
		_ = tx
	}

	return &msg
}
