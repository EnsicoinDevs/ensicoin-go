package network

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/EnsicoinDevs/eccd/utils"
	"io"
)

type Outpoint struct {
	Hash  utils.Hash
	Index uint32
}

func (outpoint *Outpoint) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	err := WriteHash(buf, &outpoint.Hash)
	if err != nil {
		return nil, err
	}

	err = WriteUint32(buf, outpoint.Index)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (outpoint *Outpoint) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)

	hashPtr, err := ReadHash(buf)
	if err != nil {
		return err
	}

	outpoint.Hash = *hashPtr

	outpoint.Index, err = ReadUint32(buf)
	return err
}

func readOutpoint(reader io.Reader) (*Outpoint, error) {
	hash, err := ReadHash(reader)
	if err != nil {
		return nil, err
	}

	index, err := ReadUint32(reader)
	if err != nil {
		return nil, err
	}

	return &Outpoint{
		Hash:  *hash,
		Index: index,
	}, nil
}

func writeOutpoint(writer io.Writer, outpoint *Outpoint) error {
	err := WriteHash(writer, &outpoint.Hash)
	if err != nil {
		return err
	}

	err = WriteUint32(writer, outpoint.Index)
	return err
}

type TxIn struct {
	PreviousOutput *Outpoint
	Script         []byte
}

func readTxIn(reader io.Reader) (*TxIn, error) {
	outpoint, err := readOutpoint(reader)
	if err != nil {
		return nil, err
	}

	scriptLength, err := ReadVarUint(reader)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, scriptLength)
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		return nil, err
	}

	return &TxIn{
		PreviousOutput: outpoint,
		Script:         buf,
	}, nil
}

func writeTxIn(writer io.Writer, txIn *TxIn) error {
	err := writeOutpoint(writer, txIn.PreviousOutput)
	if err != nil {
		return err
	}

	err = WriteVarUint(writer, uint64(len(txIn.Script)))
	if err != nil {
		return err
	}

	_, err = writer.Write(txIn.Script)
	return err
}

type TxOut struct {
	Value  uint64
	Script []byte
}

func (txOut *TxOut) String() string {
	return fmt.Sprintf("TxOut[Value: %d, Script: %v]", txOut.Value, txOut.Script)
}

func readTxOut(reader io.Reader) (*TxOut, error) {
	value, err := ReadUint64(reader)
	if err != nil {
		return nil, err
	}

	scriptLength, err := ReadVarUint(reader)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, scriptLength)
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		return nil, err
	}

	return &TxOut{
		Value:  value,
		Script: buf,
	}, nil
}

func writeTxOut(writer io.Writer, txOut *TxOut) error {
	err := WriteUint64(writer, txOut.Value)
	if err != nil {
		return err
	}

	err = WriteVarUint(writer, uint64(len(txOut.Script)))
	if err != nil {
		return err
	}

	_, err = writer.Write(txOut.Script)
	return err
}

type TxMessage struct {
	Version uint32
	Flags   []string
	Inputs  []*TxIn
	Outputs []*TxOut
}

func NewTxMessage() *TxMessage {
	return &TxMessage{}
}

func (msg *TxMessage) Decode(reader io.Reader) error {
	version, err := ReadUint32(reader)
	if err != nil {
		return err
	}

	flagsCount, err := ReadVarUint(reader)
	if err != nil {
		return err
	}

	var flags []string

	for i := uint64(0); i < flagsCount; i++ {
		flag, err := ReadVarString(reader)
		if err != nil {
			return err
		}

		flags = append(flags, flag)
	}

	inputsCount, err := ReadVarUint(reader)
	if err != nil {
		return err
	}

	var inputs []*TxIn

	for i := uint64(0); i < inputsCount; i++ {
		txIn, err := readTxIn(reader)
		if err != nil {
			return err
		}

		inputs = append(inputs, txIn)
	}

	outputsCount, err := ReadVarUint(reader)
	if err != nil {
		return nil
	}

	var outputs []*TxOut

	for i := uint64(0); i < outputsCount; i++ {
		txOut, err := readTxOut(reader)
		if err != nil {
			return nil
		}

		outputs = append(outputs, txOut)
	}

	msg.Version = version
	msg.Flags = flags
	msg.Inputs = inputs
	msg.Outputs = outputs

	return nil
}

func (msg *TxMessage) Encode(writer io.Writer) error {
	err := WriteUint32(writer, msg.Version)
	if err != nil {
		return err
	}

	err = WriteVarUint(writer, uint64(len(msg.Flags)))
	if err != nil {
		return err
	}

	for _, flag := range msg.Flags {
		err = WriteVarString(writer, flag)
		if err != nil {
			return err
		}
	}

	err = WriteVarUint(writer, uint64(len(msg.Inputs)))
	if err != nil {
		return err
	}

	for _, input := range msg.Inputs {
		err = writeTxIn(writer, input)
		if err != nil {
			return err
		}
	}

	err = WriteVarUint(writer, uint64(len(msg.Outputs)))
	if err != nil {
		return err
	}

	for _, output := range msg.Outputs {
		err = writeTxOut(writer, output)
		if err != nil {
			return err
		}
	}

	return nil
}

func (msg *TxMessage) Hash() *utils.Hash {
	buf := bytes.NewBuffer(nil)
	_ = msg.Encode(buf)

	hash := utils.Hash(sha256.Sum256(buf.Bytes()))

	hash = sha256.Sum256(hash[:])

	return &hash
}

func (msg *TxMessage) SHash(input *TxIn, value uint64) *utils.Hash {
	buf := bytes.NewBuffer(nil)
	_ = WriteUint32(buf, msg.Version)
	_ = WriteVarUint(buf, uint64(len(msg.Flags)))
	for _, flag := range msg.Flags {
		_ = WriteVarString(buf, flag)
	}

	bufOutpoints := bytes.NewBuffer(nil) // TODO: optimize

	for _, input := range msg.Inputs {
		_ = writeOutpoint(buf, input.PreviousOutput)
	}

	hash := sha256.Sum256(bufOutpoints.Bytes())

	hash = sha256.Sum256(hash[:])

	_, _ = buf.Write(hash[:])

	writeOutpoint(buf, input.PreviousOutput)

	_ = WriteUint64(buf, value)

	bufOutputs := bytes.NewBuffer(nil) // TODO: same

	for _, output := range msg.Outputs {
		_ = writeTxOut(bufOutputs, output)
	}

	hash = sha256.Sum256(bufOutputs.Bytes())

	hash = sha256.Sum256(hash[:])

	_, _ = buf.Write(hash[:])

	shash := utils.Hash(sha256.Sum256(buf.Bytes()))

	shash = sha256.Sum256(hash[:])

	return &shash
}

func (msg *TxMessage) MsgType() string {
	return "tx"
}

func (msg TxMessage) String() string {
	return fmt.Sprintf("TxMessage[Version: %d, Flags: %v, Inputs: %v, Outputs: %v]", msg.Version, msg.Flags, msg.Inputs, msg.Outputs)
}
