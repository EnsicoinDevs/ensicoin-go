package blockchain

import (
	"bytes"
	"github.com/EnsicoinDevs/eccd/network"
	"io"
)

type UtxoEntry struct {
	amount      uint64
	script      []byte
	blockHeight uint32

	coinBase bool
}

func newUtxoEntry() *UtxoEntry {
	return &UtxoEntry{}
}

func WriteUtxoEntry(writer io.Writer, entry *UtxoEntry) error {
	err := network.WriteUint64(writer, entry.amount)
	if err != nil {
		return err
	}

	err = network.WriteVarUint(writer, uint64(len(entry.script)))
	if err != nil {
		return err
	}

	_, err = writer.Write(entry.script)
	if err != nil {
		return err
	}

	var coinBase byte
	if entry.coinBase {
		coinBase = 0x01
	}
	_, err = writer.Write([]byte{coinBase})
	if err != nil {
		return err
	}

	return nil
}

func ReadUtxoEntry(reader io.Reader) (*UtxoEntry, error) {
	entry := new(UtxoEntry)

	var err error

	entry.amount, err = network.ReadUint64(reader)
	if err != nil {
		return nil, err
	}

	scriptLength, err := network.ReadVarUint(reader)
	if err != nil {
		return nil, err
	}

	entry.script = make([]byte, scriptLength)
	_, err = io.ReadFull(reader, entry.script)

	coinBase := make([]byte, 1)
	_, err = reader.Read(coinBase)
	if err != nil {
		return nil, err
	}

	if coinBase[0] != 0x00 {
		entry.coinBase = true
	}

	return entry, nil
}

func (entry *UtxoEntry) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	err := WriteUtxoEntry(buf, entry)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (entry *UtxoEntry) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	newEntry, err := ReadUtxoEntry(buf)

	*entry = *newEntry

	return err
}

func (entry *UtxoEntry) Amount() uint64 {
	return entry.amount
}

func (entry *UtxoEntry) Script() []byte {
	return entry.script
}

func (entry *UtxoEntry) BlockHeight() uint32 {
	return entry.blockHeight
}

func (entry *UtxoEntry) IsCoinBase() bool {
	return entry.coinBase
}

type Utxos struct {
	entries map[network.Outpoint]*UtxoEntry
}

func newUtxos() *Utxos {
	return &Utxos{
		entries: make(map[network.Outpoint]*UtxoEntry),
	}
}

func (utxos *Utxos) AddEntry(outpoint *network.Outpoint, entry *UtxoEntry) {
	utxos.entries[*outpoint] = entry
}

func (utxos *Utxos) AddEntryWithTx(outpoint *network.Outpoint, tx *Tx, blockHeight uint32) {
	output := tx.Msg.Outputs[outpoint.Index]

	utxos.AddEntry(outpoint, &UtxoEntry{
		amount:      output.Value,
		script:      output.Script,
		blockHeight: blockHeight,
		coinBase:    tx.IsCoinBase(),
	})
}

func (utxos *Utxos) FindEntry(outpoint *network.Outpoint) *UtxoEntry {
	return utxos.entries[*outpoint]
}

func (utxos *Utxos) RemoveEntry(outpoint *network.Outpoint) *UtxoEntry {
	entry := utxos.FindEntry(outpoint)

	delete(utxos.entries, *outpoint)

	return entry
}
