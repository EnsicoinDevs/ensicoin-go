package sssync

import (
	"encoding/hex"
	"github.com/EnsicoinDevs/eccd/blockchain"
	"github.com/EnsicoinDevs/eccd/mempool"
	"github.com/EnsicoinDevs/eccd/network"
	"github.com/EnsicoinDevs/eccd/peer"
	"github.com/EnsicoinDevs/eccd/utils"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
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
	log.Debug("sssync shutting down")
	defer log.Debug("sssync shutdown complete")

	close(sync.quit)

	return nil
}

func (sync *Synchronizer) OnPushedBlock(block *blockchain.Block) error {
	log.WithFields(log.Fields{
		"hash":   hex.EncodeToString(block.Msg.Header.Hash()[:]),
		"height": block.Msg.Header.Height,
	}).Info("best block updated")

	log.Debug("removing tx from mempool")

	for _, tx := range block.Txs[1:] {
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

	return nil
}

func (sync *Synchronizer) OnPoppedBlock(block *blockchain.Block) error {
	log.WithFields(log.Fields{
		"hash":   hex.EncodeToString(block.Msg.Header.Hash()[:]),
		"height": block.Msg.Header.Height,
	}).Info("best block updated")

	return sync.handlePoppedBlock(block)
}

func (sync *Synchronizer) HandleBlockInvVect(peer *peer.Peer, invVect *network.InvVect) error {
	if invVect.InvType != network.INV_VECT_BLOCK {
		panic("the invtype of the invvect is not block")
	}

	haveBlock, err := sync.blockchain.HaveBlock(invVect.Hash)
	if err != nil {
		return err
	}

	if !haveBlock {
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
	log.Debug("processing a block")
	start := time.Now()
	valid, err := sync.blockchain.ProcessBlock(blockchain.NewBlockFromBlockMessage(message))
	if err != nil {
		log.WithError(err).WithField("hash", hex.EncodeToString(message.Header.Hash()[:])).Error("error processing a block")
		return
	}
	if valid != nil {
		log.WithField("hash", hex.EncodeToString(message.Header.Hash()[:])).WithField("error", valid).Info("processed an invalid block")
		return
	}
	log.Debugf("time elapsed: %s", time.Since(start))
}

func (sync *Synchronizer) HandleReadyPeer(peer *peer.Peer) {
	log.WithFields(log.Fields{
		"func":  "HandleReadyPeer",
		"mutex": "synchronizer",
	}).Trace("locking")
	sync.mutex.Lock()
	defer log.WithFields(log.Fields{
		"func":  "HandleReadyPeer",
		"mutex": "synchronizer",
	}).Trace("unlocking")
	defer sync.mutex.Unlock()

	if sync.synchronizingPeer == nil {
		sync.synchronizingPeer = peer

		sync.synchronize()
	}
}

func (sync *Synchronizer) HandleDisconnectedPeer(peer *peer.Peer) {
	log.WithFields(log.Fields{
		"func":  "HandleDisconnectedPeer",
		"mutex": "synchronizer",
	}).Trace("locking")
	sync.mutex.Lock()
	defer log.WithFields(log.Fields{
		"func":  "HandleDisconnectedPeer",
		"mutex": "synchronizer",
	}).Trace("unlocking")
	defer sync.mutex.Unlock()

	if sync.synchronizingPeer == peer {
		sync.synchronizingPeer = nil
	}
}

func (sync *Synchronizer) synchronize() {
	blockLocator, err := sync.blockchain.GenerateBlockLocator()
	if err != nil {
		log.WithError(err).Error("error generating the block locator")
		return
	}

	sync.synchronizingPeer.Send(&network.GetBlocksMessage{
		BlockLocator: blockLocator,
		HashStop:     utils.NewHash(nil),
	})
}

func (sync *Synchronizer) handlePoppedBlock(block *blockchain.Block) error {
	for _, tx := range block.Txs {
		sync.mempool.ProcessTx(tx)
	}

	return nil
}
