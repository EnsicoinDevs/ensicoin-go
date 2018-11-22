package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"strconv"
)

type Outpoint struct {
	Hash  string
	Index uint32
}

type TxIn struct {
	PreviousOutput Outpoint
	Script         []byte
}

type TxOut struct {
	Value  uint64
	Script []byte
}

type Tx struct {
	Hash    string
	Version uint32
	Flags   []string
	Inputs  []*TxIn
	Outputs []*TxOut
}

func NewTxFromTxMessage(txMsg *network.TxMessage) *Tx {
	tx := Tx{
		Version: txMsg.Version,
		Flags:   txMsg.Flags,
	}

	for _, input := range txMsg.Inputs {
		tx.Inputs = append(tx.Inputs, &TxIn{
			PreviousOutput: Outpoint{
				Hash:  input.PreviousOutput.Hash,
				Index: input.PreviousOutput.Index,
			},
			Script: input.Script,
		})
	}

	for _, output := range txMsg.Outputs {
		tx.Outputs = append(tx.Outputs, &TxOut{
			Value:  output.Value,
			Script: output.Script,
		})
	}

	return &tx
}

func (tx *Tx) ComputeHash() {
	h := sha256.New()

	// TODO: hash

	//h.Write([]byte(strconv.Itoa(tx.Version)))
	//h.Write([]byte(strings.Join(tx.Flags, "")))

	for _, input := range tx.Inputs {
		//h.Write([]byte(input.PreviousOutput.TxHash))
		h.Write([]byte(strconv.FormatUint(uint64(input.PreviousOutput.Index), 10)))
	}

	for _, output := range tx.Outputs {
		h.Write([]byte(strconv.FormatUint(output.Value, 10)))
		h.Write(output.Script)
	}

	v := h.Sum(nil)

	h.Reset()

	h.Write(v)

	tx.Hash = hex.EncodeToString(h.Sum(nil))
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
