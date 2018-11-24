package blockchain

import (
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"time"
)

var genesisBlock = Block{
	Msg: &network.BlockMessage{
		Header: &network.BlockHeader{
			Version:   0,
			Flags:     []string{"ici cest limag"},
			Timestamp: time.Date(2019, 8, 26, 23, 42, 0, 0, time.UTC),
			Bits:      1,
			Nonce:     42,
		},
	},
}
