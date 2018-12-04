package miner

import (
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"github.com/EnsicoinDevs/ensicoincoin/scripts"
)

func newCoinbaseTx(value uint64) *network.TxMessage {
	return &network.TxMessage{
		Version: 0,
		Flags:   []string{"this_tx_is_useless"},
		Inputs:  nil,
		Outputs: []*network.TxOut{&network.TxOut{
			Value:  value,
			Script: []byte{byte(scripts.OP_TRUE)},
		}},
	}
}
