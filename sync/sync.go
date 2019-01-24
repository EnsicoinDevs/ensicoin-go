package sync

import (
	"github.com/EnsicoinDevs/ensicoincoin/blockchain"
	"github.com/EnsicoinDevs/ensicoincoin/mempool"
	"github.com/EnsicoinDevs/ensicoincoin/miner"
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"github.com/EnsicoinDevs/ensicoincoin/peer"
	log "github.com/sirupsen/logrus"
)

type Synchronizer struct {
	Blockchain   *blockchain.Blockchain
	Mempool      *mempool.Mempool
	PeersManager *peer.PeersManager
	Miner        *miner.Miner

	quit chan struct{}
}

func NewSynchronizer(blockchain *blockchain.Blockchain, mempool *mempool.Mempool, peersManager *peer.PeersManager, miner *miner.Miner) *Synchronizer {
	return &Synchronizer{
		Blockchain:   blockchain,
		Mempool:      mempool,
		PeersManager: peersManager,
		Miner:        miner,

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

func (sync *Synchronizer) ProcessBlock(message *network.BlockMessage) {
	valid, err := sync.Blockchain.ProcessBlock(blockchain.NewBlockFromBlockMessage(message))
	if err != nil {
		log.WithError(err).WithField("block", message).Error("error processing a block")
		return
	}
	if valid != nil {
		log.WithField("block", message).WithField("error", valid).Info("processed an invalid block")
		return
	}

	log.WithField("block", message).Info("processed a valid block")
}

func (sync *Synchronizer) handlePushedBlock(block *blockchain.Block) {
	for _, tx := range block.Txs {
		sync.Mempool.RemoveTx(tx)
	}

	sync.PeersManager.Broadcast(block.Msg)
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
