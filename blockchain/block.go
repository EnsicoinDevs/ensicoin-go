package blockchain

import (
	"bytes"
	"errors"
	"github.com/EnsicoinDevs/eccd/network"
	"github.com/EnsicoinDevs/eccd/utils"
	"math/big"
	"reflect"
)

var MaxTarget = new(big.Int).Lsh(big.NewInt(1), 256)

type Block struct {
	Msg *network.BlockMessage
	Txs []*Tx
}

func NewBlock() *Block {
	return &Block{
		Msg: network.NewBlockMessage(),
	}
}

func NewBlockFromBlockMessage(msg *network.BlockMessage) *Block {
	block := Block{
		Msg: msg,
	}

	for _, tx := range msg.Txs {
		block.Txs = append(block.Txs, NewTxFromTxMessage(tx))
	}

	return &block
}

func (block *Block) Hash() *utils.Hash {
	return block.Msg.Header.Hash()
}

func (block *Block) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := block.Msg.Encode(buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (block *Block) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	err := block.Msg.Decode(buf)
	if err != nil {
		return err
	}

	for _, tx := range block.Msg.Txs {
		block.Txs = append(block.Txs, NewTxFromTxMessage(tx))
	}

	return nil
}

func (block *Block) IsSane() error {
	// header

	if !CheckProofOfWork(block.Msg.Header) {
		return errors.New("bad proof of work")
	}

	// txs

	if len(block.Txs) == 0 {
		return errors.New("no txs")
	}

	if !block.Txs[0].IsCoinBase() {
		return errors.New("first tx of the block is not a coinbase")
	}

	for _, tx := range block.Txs[1:] {
		if tx.IsCoinBase() {
			return errors.New("the block have more than one coinbase")
		}
	}

	for _, tx := range block.Txs {
		if !tx.IsSane() {
			return errors.New("a tx of the block is not sane")
		}
	}

	var merkleHashes []*utils.Hash
	for _, tx := range block.Txs {
		merkleHashes = append(merkleHashes, tx.Hash())
	}

	merkleRoot := ComputeMerkleRoot(merkleHashes)
	if !reflect.DeepEqual(merkleRoot, block.Msg.Header.HashMerkleRoot) {
		return errors.New("the merkle root is not correct")
	}

	seenTxHashes := make(map[*utils.Hash]struct{})
	for _, tx := range block.Txs {
		if _, exists := seenTxHashes[tx.Hash()]; exists {
			return errors.New("a tx is a duplicate")
		}

		seenTxHashes[tx.Hash()] = struct{}{}
	}

	return nil
}

func (block *Block) CalcBlockSubsidy() uint64 {
	if block.Msg.Header.Height == 0 {
		return 0
	}

	if block.Msg.Header.Height == 1 {
		return uint64(0x3ffff)
	}

	return uint64(0x200000000000) >> ((block.Msg.Header.Height - 1) / 0x40000)
}

func (block *Block) GetWork() *big.Int {
	return new(big.Int).Sub(MaxTarget, block.Msg.Header.Target)
}

func CheckProofOfWork(header *network.BlockHeader) bool {
	return !(header.Hash().Big().Cmp(header.Target) > 0)
}
