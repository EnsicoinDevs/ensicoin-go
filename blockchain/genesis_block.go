package blockchain

import (
	"github.com/EnsicoinDevs/eccd/network"
	"github.com/EnsicoinDevs/eccd/utils"
	"math/big"
	"time"
)

func init() {
	genesisBlock.Msg.Header.Target, _ = big.NewInt(0).SetString("0000f0000000000000000000000000", 16)
}

const FT byte = 0x00

var genesisBlock = Block{
	Msg: &network.BlockMessage{
		Header: &network.BlockHeader{
			Version:        0,
			Flags:          []string{"ici cest limag"},
			Timestamp:      time.Unix(1566862920, 0),
			HashPrevBlock:  utils.NewHash(make([]byte, 32)),
			HashMerkleRoot: utils.NewHash(make([]byte, 32)),
			Nonce:          42,
		},
	},
}

var GenesisBlock = genesisBlock
