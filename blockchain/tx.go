package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"strconv"
)

type Tx struct {
	Msg *network.TxMessage
}

func (tx *Tx) Hash() []byte {
	return tx.Msg.Hash()
}

// Validate applies the validation tests of a transaction and returns true if they pass, false if they fail.
func (tx *Tx) IsSane() bool {
	if len(tx.Outputs) == 0 {
		return false
	}

	// TODO: check for the maximum allowed size

	for _, output := range tx.Outputs {
		if output.Value < 0 {
			return false
		}
	}

	return true
}

// IsCoinBase returns true if the transaction is a coinbase.
func (tx *Tx) IsCoinBase() bool {
	return len(tx.Inputs) == 0
}

func (tx *Tx) ToTxMessage() *network.TxMessage {
	msg := network.TxMessage{
		Version: tx.Version,
		Flags:   tx.Flags,
	}

	for _, input := range tx.Inputs {
		msg.Inputs = append(msg.Inputs, &network.TxIn{
			PreviousOutput: &network.Outpoint{
				Hash:  input.PreviousOutput.Hash,
				Index: input.PreviousOutput.Index,
			},
			Script: input.Script,
		})
	}

	for _, output := range tx.Outputs {
		msg.Outputs = append(msg.Outputs, &network.TxOut{
			Value:  output.Value,
			Script: output.Script,
		})
	}

	return &msg
}
