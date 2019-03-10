package sssync

import (
	"encoding/hex"
	"github.com/EnsicoinDevs/ensicoincoin/blockchain"
	"github.com/EnsicoinDevs/ensicoincoin/mempool"
	"github.com/EnsicoinDevs/ensicoincoin/miner"
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"github.com/EnsicoinDevs/ensicoincoin/peer"
	"github.com/EnsicoinDevs/ensicoincoin/utils"
	log "github.com/sirupsen/logrus"
	"sync"
)

type Config struct {
	Broadcast       func(network.Message) error
	OnAcceptedBlock func(*blockchain.Block) error
}

type Synchronizer struct {
	config Config

	Blockchain *blockchain.Blockchain
	Mempool    *mempool.Mempool
	Miner      *miner.Miner

	mutex             *sync.Mutex
	synchronizingPeer *peer.Peer

	quit chan struct{}
}

func NewSynchronizer(config *Config, blockchain *blockchain.Blockchain, mempool *mempool.Mempool, miner *miner.Miner) *Synchronizer {
	return &Synchronizer{
		config: *config,

		Blockchain: blockchain,
		Mempool:    mempool,
		Miner:      miner,

		mutex: &sync.Mutex{},

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
				go sync.handlePushedBlock(block)
			case block := <-sync.Blockchain.PoppedBlocks:
				go sync.handlePoppedBlock(block)
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
		peer.Send(&network.GetDataMessage{
			Inventory: []*network.InvVect{invVect},
		})
	}

	return nil
}

func (sync *Synchronizer) HandleBlock(peer *peer.Peer, message *network.BlockMessage) {
	sync.ProcessBlock(message)
}

func (sync *Synchronizer) ProcessBlock(message *network.BlockMessage) {
	valid, err := sync.Blockchain.ProcessBlock(blockchain.NewBlockFromBlockMessage(message))
	if err != nil {
		log.WithError(err).WithField("hash", hex.EncodeToString(message.Header.Hash()[:])).Error("error processing a block")
		return
	}
	if valid != nil {
		log.WithField("hash", hex.EncodeToString(message.Header.Hash()[:])).WithField("error", valid).Info("processed an invalid block")
		return
	}

	log.WithField("hash", hex.EncodeToString(message.Header.Hash()[:])).Info("processed a valid block")
}

func (sync *Synchronizer) HandleReadyPeer(peer *peer.Peer) {
	sync.mutex.Lock()
	defer sync.mutex.Unlock()

	if sync.synchronizingPeer == nil {
		sync.synchronizingPeer = peer

		sync.synchronize()
	}
}

func (sync *Synchronizer) HandleDisconnectedPeer(peer *peer.Peer) {
	sync.mutex.Lock()
	defer sync.mutex.Unlock()

	if sync.synchronizingPeer == peer {
		sync.synchronizingPeer = nil
	}
}

func (sync *Synchronizer) synchronize() {
	longestChain, err := sync.Blockchain.FindLongestChain()
	if err != nil {
		log.WithError(err).Error("error finding the longest chain")
		return
	}

	sync.synchronizingPeer.Send(&network.GetBlocksMessage{
		BlockLocator: []*utils.Hash{longestChain.Hash()},
		HashStop:     utils.NewHash(nil),
	})
}

func (sync *Synchronizer) handlePushedBlock(block *blockchain.Block) {
	for _, tx := range block.Txs {
		sync.Mempool.RemoveTx(tx)
	}

	sync.config.Broadcast(&network.InvMessage{
		Inventory: []*network.InvVect{
			&network.InvVect{
				InvType: network.INV_VECT_BLOCK,
				Hash:    block.Hash(),
			},
		},
	})
	sync.updateMinerBestBlock()
}

func (sync *Synchronizer) handlePoppedBlock(block *blockchain.Block) {
	for _, tx := range block.Txs {
		sync.Mempool.ProcessTx(tx)
	}

	sync.updateMinerBestBlock()
}

func (sync *Synchronizer) updateMinerBestBlock() error {
	log.Debug("updating miner best block")

	bestBlock, err := sync.Blockchain.FindLongestChain()
	if err != nil {
		return err
	}

	log.Debug(bestBlock)

	if sync.Miner.BestBlock == nil {
		log.Debug("miner best block is nil")

		sync.Miner.UpdateBestBlock(bestBlock)
		return nil
	}

	if !sync.Miner.BestBlock.Hash().IsEqual(bestBlock.Hash()) {
		log.Debug("miner best block is outdated")

		sync.Miner.UpdateBestBlock(bestBlock)
		return nil
	}

	return nil
}
