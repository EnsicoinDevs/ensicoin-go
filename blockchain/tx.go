package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

type TxInput struct {
	PreviousOutput struct {
		TxHash string
		Index  int
	}
	PreviousTxHash      string
	PreviousOutputIndex int
	Script              []string
}

type TxOutput struct {
	Value  uint64
	Script []string
}

type Tx struct {
	Hash    string
	Version int
	Flags   []string
	Inputs  []*TxInput
	Outputs []*TxOutput
}

func (tx *Tx) ComputeHash() {
	h := sha256.New()

	h.Write([]byte(strconv.Itoa(tx.Version)))
	h.Write([]byte(strings.Join(tx.Flags, "")))

	for _, input := range tx.Inputs {
		h.Write([]byte(input.PreviousTxHash))
		h.Write([]byte(strconv.Itoa(input.PreviousOutputIndex)))
	}

	for _, output := range tx.Outputs {
		h.Write([]byte(strconv.FormatUint(output.Value, 10)))
		h.Write([]byte(strings.Join(output.Script, "")))
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
