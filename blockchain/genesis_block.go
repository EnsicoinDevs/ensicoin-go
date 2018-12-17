package blockchain

import (
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"github.com/EnsicoinDevs/ensicoincoin/utils"
	"time"
)

var genesisBlock = Block{
	Msg: &network.BlockMessage{
		Header: &network.BlockHeader{
			Version:        0,
			Flags:          []string{"ici cest limag"},
			Timestamp:      time.Date(2019, 8, 26, 23, 42, 0, 0, time.UTC),
			HashPrevBlock:  utils.NewHash([]byte("olala")),
			HashMerkleRoot: utils.NewHash([]byte("olali")),
			Bits:           0x1effffff,
			Nonce:          42,
		},
	},
}
