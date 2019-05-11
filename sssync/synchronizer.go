package sssync

import (
	"encoding/hex"
	"github.com/EnsicoinDevs/ensicoincoin/blockchain"
	"github.com/EnsicoinDevs/ensicoincoin/mempool"
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"github.com/EnsicoinDevs/ensicoincoin/peer"
	"github.com/EnsicoinDevs/ensicoincoin/utils"
	log "github.com/sirupsen/logrus"
	"sync"
)

type Config struct {
	Broadcast func(network.Message) error
}

type Synchronizer struct {
	config Config

	blockchain *blockchain.Blockchain
	mempool    *mempool.Mempool

	mutex             *sync.Mutex
	synchronizingPeer *peer.Peer

	quit chan struct{}
}

func NewSynchronizer(config *Config, blockchain *blockchain.Blockchain, mempool *mempool.Mempool) *Synchronizer {
	return &Synchronizer{
		config: *config,

		blockchain: blockchain,
		mempool:    mempool,

		mutex: &sync.Mutex{},

		quit: make(chan struct{}),
	}
}

func (sync *Synchronizer) Start() error {
	return nil
}

func (sync *Synchronizer) Stop() error {
	return nil
}

func (sync *Synchronizer) OnPushedBlock(block *blockchain.Block) error {
	log.WithFields(log.Fields{
		"hash":   hex.EncodeToString(block.Msg.Header.Hash()[:]),
		"height": block.Msg.Header.Height,
	}).Info("block pushed")

	return sync.handlePushedBlock(block)
}

func (sync *Synchronizer) OnPoppedBlock(block *blockchain.Block) error {
	log.WithFields(log.Fields{
		"hash":   hex.EncodeToString(block.Msg.Header.Hash()[:]),
		"height": block.Msg.Header.Height,
	}).Info("block popped")

	return sync.handlePoppedBlock(block)
}

func (sync *Synchronizer) HandleBlockInvVect(peer *peer.Peer, invVect *network.InvVect) error {
	if invVect.InvType != network.INV_VECT_BLOCK {
		panic("the invtype of the invvect is not block")
	}

	block, err := sync.blockchain.FindBlockByHash(invVect.Hash)
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
	valid, err := sync.blockchain.ProcessBlock(blockchain.NewBlockFromBlockMessage(message))
	if err != nil {
		log.WithError(err).WithField("hash", hex.EncodeToString(message.Header.Hash()[:])).Error("error processing a block")
		return
	}
	if valid != nil {
		log.WithField("hash", hex.EncodeToString(message.Header.Hash()[:])).WithField("error", valid).Info("processed an invalid block")
		return
	}
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
	bestBlock, err := sync.blockchain.FindBestBlock()
	if err != nil {
		log.WithError(err).Error("error finding the longest chain")
		return
	}

	sync.synchronizingPeer.Send(&network.GetBlocksMessage{
		BlockLocator: []*utils.Hash{bestBlock.Hash()},
		HashStop:     utils.NewHash(nil),
	})
}

func (sync *Synchronizer) handlePushedBlock(block *blockchain.Block) error {
	log.Debug("removing tx from mempool")

	for _, tx := range block.Txs {
		sync.mempool.RemoveTx(tx)
	}

	log.Debug("broadcasting")

	sync.config.Broadcast(&network.InvMessage{
		Inventory: []*network.InvVect{
			&network.InvVect{
				InvType: network.INV_VECT_BLOCK,
				Hash:    block.Hash(),
			},
		},
	})

	log.Debug("updtating best block")

	return nil
}

func (sync *Synchronizer) handlePoppedBlock(block *blockchain.Block) error {
	for _, tx := range block.Txs {
		sync.mempool.ProcessTx(tx)
	}

	return nil
}
