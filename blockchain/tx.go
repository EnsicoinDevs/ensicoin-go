package blockchain

import (
	"github.com/EnsicoinDevs/eccd/network"
	"github.com/EnsicoinDevs/eccd/utils"
)

type Tx struct {
	Msg *network.TxMessage
}

func NewTxFromTxMessage(msg *network.TxMessage) *Tx {
	return &Tx{
		Msg: msg,
	}
}

func (tx *Tx) Hash() *utils.Hash {
	return tx.Msg.Hash()
}

func (tx *Tx) SHash(input *network.TxIn, value uint64) *utils.Hash {
	return tx.SHash(input, value)
}

// IsSane applies the validation tests of a transaction and returns true if they pass, false if they fail.
func (tx *Tx) IsSane() bool {
	if len(tx.Msg.Outputs) == 0 {
		return false
	}

	seenInputs := make(map[*network.Outpoint]struct{})
	for _, input := range tx.Msg.Inputs {
		if _, exists := seenInputs[input.PreviousOutput]; exists {
			return false
		}

		seenInputs[input.PreviousOutput] = struct{}{}
	}

	return true
}

// IsCoinBase returns true if the transaction is a coinbase.
func (tx *Tx) IsCoinBase() bool {
	return len(tx.Msg.Inputs) == 0
}
