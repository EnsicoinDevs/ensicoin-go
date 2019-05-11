package blockchain

import (
	"github.com/EnsicoinDevs/eccd/network"
	"github.com/EnsicoinDevs/eccd/utils"
	"time"
)

const FT byte = 0x00

var genesisBlock = Block{
	Msg: &network.BlockMessage{
		Header: &network.BlockHeader{
			Version:        0,
			Flags:          []string{"ici cest limag"},
			Timestamp:      time.Unix(1566862920, 0),
			HashPrevBlock:  utils.NewHash(make([]byte, 32)),
			HashMerkleRoot: utils.NewHash(make([]byte, 32)),
			Bits:           0x1e00f000,
			Nonce:          42,
		},
	},
}

var GenesisBlock = genesisBlock
