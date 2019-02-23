package sssync

import (
	"encoding/hex"
	"github.com/EnsicoinDevs/ensicoincoin/blockchain"
	"github.com/EnsicoinDevs/ensicoincoin/mempool"
	"github.com/EnsicoinDevs/ensicoincoin/miner"
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"github.com/EnsicoinDevs/ensicoincoin/peer"
	log "github.com/sirupsen/logrus"
)

type Synchronizer struct {
	Blockchain *blockchain.Blockchain
	Mempool    *mempool.Mempool
	Miner      *miner.Miner

	synchronizingPeer *peer.Peer

	quit chan struct{}
}

func NewSynchronizer(blockchain *blockchain.Blockchain, mempool *mempool.Mempool, miner *miner.Miner) *Synchronizer {
	return &Synchronizer{
		Blockchain: blockchain,
		Mempool:    mempool,
		Miner:      miner,

		quit: make(chan struct{}),
	}
}

func (sync *Synchronizer) Start() {
	go func() {
		for {
			select {
			case <-sync.quit:
				return
			case block := <-sync.Blockchain.PushedBlocks:
				sync.handlePushedBlock(block)
			case block := <-sync.Blockchain.PoppedBlocks:
				sync.handlePoppedBlock(block)
			}
		}
	}()
}

func (sync *Synchronizer) Stop() {
	sync.quit <- struct{}{}
}

func (sync *Synchronizer) HandleBlockInvVect(peer *peer.Peer, invVect *network.InvVect) error {
	if invVect.InvType != network.INV_VECT_BLOCK {
		panic("the invtype of the invvect is not block")
	}

	block, err := sync.Blockchain.FindBlockByHash(invVect.Hash)
	if err != nil {
		return err
	}

	if block == nil {
	}

	return nil
}

func (sync *Synchronizer) HandleBlock(peer *peer.Peer, message *network.BlockMessage) {
	sync.Blockchain.ProcessBlock(blockchain.NewBlockFromBlockMessage(message))
}

func (sync *Synchronizer) HandleReadyPeer(peer *peer.Peer) {

}

func (sync *Synchronizer) HandleDisconnectedPeer(peer *peer.Peer) {

}

func (sync *Synchronizer) ProcessBlock(message *network.BlockMessage) {
	valid, err := sync.Blockchain.ProcessBlock(blockchain.NewBlockFromBlockMessage(message))
	if err != nil {
		log.WithError(err).WithField("block_hash", hex.EncodeToString(message.Header.Hash()[:])).Error("error processing a block")
		return
	}
	if valid != nil {
		log.WithField("block_hash", hex.EncodeToString(message.Header.Hash()[:])).WithField("error", valid).Info("processed an invalid block")
		return
	}

	log.WithField("blockHash", hex.EncodeToString(message.Header.Hash()[:])).Info("processed a valid block")
}

func (sync *Synchronizer) handlePushedBlock(block *blockchain.Block) {
	for _, tx := range block.Txs {
		sync.Mempool.RemoveTx(tx)
	}

	sync.updateMinerBestBlock()
}

func (sync *Synchronizer) handlePoppedBlock(block *blockchain.Block) {
	for _, tx := range block.Txs {
		sync.Mempool.ProcessTx(tx)
	}

	sync.updateMinerBestBlock()
}

func (sync *Synchronizer) updateMinerBestBlock() error {
	bestBlock, err := sync.Blockchain.FindLongestChain()
	if err != nil {
		return err
	}

	if sync.Miner.BestBlock == nil {
		sync.Miner.UpdateBestBlock(bestBlock)
		return nil
	}

	if !sync.Miner.BestBlock.Hash().IsEqual(bestBlock.Hash()) {
		sync.Miner.UpdateBestBlock(bestBlock)
		return nil
	}

	return nil
}
