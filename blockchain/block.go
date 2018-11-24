package blockchain

import (
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"github.com/EnsicoinDevs/ensicoincoin/utils"
	"math/big"
)

type Block struct {
	Msg *network.BlockMessage
	Txs []*Tx
}

func (block *Block) Hash() *utils.Hash {
	return block.Msg.Header.Hash()
}

func (block *Block) IsSane() bool {
	// header

	if !CheckProofOfWork(block.Msg.Header) {
		return false
	}

	// txs

	if len(block.Txs) == 0 {
		return false
	}

	if !block.Txs[0].IsCoinBase() {
		return false
	}

	for _, tx := range block.Txs[1:] {
		if tx.IsCoinBase() {
			return false
		}
	}

	for _, tx := range block.Txs {
		if !tx.IsSane() {
			return false
		}
	}

	// TODO: build and check the Merkle tree

	seenTxHashes := make(map[*utils.Hash]struct{})
	for _, tx := range block.Txs {
		if _, exists := seenTxHashes[tx.Hash()]; exists {
			return false
		}

		seenTxHashes[tx.Hash()] = struct{}{}
	}

	return true
}

func BitsToBig(bits uint32) *big.Int {
	mantissa := int64(bits & 0x00ffffff)
	exponent := uint(bits >> 24)

	if exponent <= 3 {
		mantissa >>= 8 * (3 - exponent)
		return big.NewInt(mantissa)
	}

	target := big.NewInt(mantissa)
	return target.Lsh(target, 8*(exponent-3))
}

func BigToBits(target *big.Int) uint32 {
	var mantissa uint32
	exponent := uint(len(target.Bytes()))

	if exponent <= 3 {
		mantissa = uint32(target.Bits()[0])
		mantissa <<= 8 * (3 - exponent)
	} else {
		targetCopy := new(big.Int).Set(target)
		mantissa = uint32(targetCopy.Rsh(targetCopy, 8*(exponent-3)).Bits()[0])
	}

	return uint32(exponent<<24) | mantissa
}

func HashToBig(hash *utils.Hash) *big.Int {
	return new(big.Int).SetBytes(hash[:])
}

func CheckProofOfWork(header *network.BlockHeader) bool {
	target := BitsToBig(header.Bits)

	hash := HashToBig(header.Hash())
	if hash.Cmp(target) > 0 {
		return false
	}

	return true
}
