package blockchain

import (
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"github.com/EnsicoinDevs/ensicoincoin/utils"
)

type Block struct {
	Msg *network.BlockMessage
	Txs []*Tx
}

func (block *Block) Hash() *utils.Hash {
	return block.Msg.Header.Hash()
}

func (block *Block) Validate() bool {
	return true
}
