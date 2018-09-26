package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

type TransactionInput struct {
	PreviousOutput          *TransactionOutput
	PreviousTransactionHash string
	PreviousOutputIndex     int
	Script                  []string
}

type TransactionOutput struct {
	Value  int
	Script []string
}

type Transaction struct {
	Hash    string
	Version int
	Flags   []string
	Inputs  []*TransactionInput
	Outputs []*TransactionOutput
}

func (tx *Transaction) ComputeHash() {
	h := sha256.New()

	h.Write([]byte(strconv.Itoa(tx.Version)))
	h.Write([]byte(strings.Join(tx.Flags, "")))

	for _, input := range tx.Inputs {
		h.Write([]byte(input.PreviousTransactionHash))
		h.Write([]byte(strconv.Itoa(input.PreviousOutputIndex)))
	}

	for _, output := range tx.Outputs {
		h.Write([]byte(strconv.Itoa(output.Value)))
		h.Write([]byte(strings.Join(output.Script, "")))
	}

	v := h.Sum(nil)

	h.Reset()

	h.Write(v)

	tx.Hash = hex.EncodeToString(h.Sum(nil))
}

// Validate applies the validation tests of a transaction and returns true if they pass, false if they fail.
func (tx *Transaction) Validate() bool {
	if tx.Version < 0 {
		return false
	}

	if len(tx.Inputs) < 1 || len(tx.Outputs) < 1 {
		return false
	}

	var outputsSum int

	for _, output := range tx.Outputs {
		if output.Value < 0 {
			return false
		}

		outputsSum += output.Value
	}

	var inputsSum int

	for _, input := range tx.Inputs {
		inputsSum += input.PreviousOutput.Value
	}

	if outputsSum >= inputsSum {
		return false
	}

	return true
}
