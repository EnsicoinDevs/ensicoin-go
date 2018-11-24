package blockchain

import (
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"github.com/EnsicoinDevs/ensicoincoin/utils"
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

// Validate applies the validation tests of a transaction and returns true if they pass, false if they fail.
func (tx *Tx) IsSane() bool {
	if len(tx.Msg.Outputs) == 0 {
		return false
	}

	for _, output := range tx.Msg.Outputs {
		if output.Value < 0 {
			return false
		}
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
