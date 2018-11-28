package blockchain

import (
	"bytes"
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"io"
)

type UtxoEntry struct {
	amount      uint64
	script      []byte
	blockHeight int

	coinBase bool
}

func newUtxoEntry() *UtxoEntry {
	return &UtxoEntry{}
}

func (entry *UtxoEntry) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	err := network.WriteUint64(buf, entry.amount)
	if err != nil {
		return nil, err
	}

	err = network.WriteVarUint(buf, uint64(len(entry.script)))
	if err != nil {
		return nil, err
	}

	_, err = buf.Write(entry.script)
	if err != nil {
		return nil, err
	}

	var coinBase uint8
	if entry.coinBase {
		coinBase = 1
	}
	err = network.WriteUint8(buf, coinBase)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (entry *UtxoEntry) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)

	var err error

	entry.amount, err = network.ReadUint64(buf)
	if err != nil {
		return err
	}

	scriptLength, err := network.ReadVarUint(buf)
	if err != nil {
		return err
	}

	entry.script = make([]byte, scriptLength)
	_, err = io.ReadFull(buf, entry.script)
	if err != nil {
		return err
	}

	coinBase, err := network.ReadUint8(buf)
	if err != nil {
		return err
	}

	entry.coinBase = true
	if coinBase == 0 {
		entry.coinBase = false
	}

	return nil
}

func (entry *UtxoEntry) Amount() uint64 {
	return entry.amount
}

func (entry *UtxoEntry) Script() []byte {
	return entry.script
}

func (entry *UtxoEntry) BlockHeight() int {
	return entry.blockHeight
}

func (entry *UtxoEntry) IsCoinBase() bool {
	return entry.coinBase
}

type Utxos struct {
	entries map[*network.Outpoint]*UtxoEntry
}

func newUtxos() *Utxos {
	return &Utxos{
		entries: make(map[*network.Outpoint]*UtxoEntry),
	}
}

func (utxos *Utxos) AddEntry(outpoint *network.Outpoint, entry *UtxoEntry) {
	utxos.entries[outpoint] = entry
}

func (utxos *Utxos) AddEntryWithTx(outpoint *network.Outpoint, tx *Tx, blockHeight int) {
	output := tx.Msg.Outputs[outpoint.Index]

	utxos.AddEntry(outpoint, &UtxoEntry{
		amount:      output.Value,
		script:      output.Script,
		blockHeight: blockHeight,
		coinBase:    tx.IsCoinBase(),
	})
}

func (utxos *Utxos) FindEntry(outpoint *network.Outpoint) *UtxoEntry {
	return utxos.entries[outpoint]
}
