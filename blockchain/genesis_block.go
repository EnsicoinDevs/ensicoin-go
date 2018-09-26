package blockchain

import (
	"time"
)

var genesisBlock = Block{
	Header: BlockHeader{
		Version:   0,
		Flags:     []string{"ici cest limag"},
		Timestamp: time.Date(2019, 8, 26, 23, 42, 0, 0, time.UTC),
		Nonce:     42,
	},
}
