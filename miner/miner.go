package miner

import (
	"fmt"
	"github.com/EnsicoinDevs/ensicoincoin/blockchain"
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"github.com/EnsicoinDevs/ensicoincoin/utils"
	log "github.com/sirupsen/logrus"
	"math"
	"math/big"
	"sync"
	"time"
)

type Config struct {
	ProcessBlock func(*network.BlockMessage)
}

type Miner struct {
	Config *Config

	Lock  sync.Mutex
	block *network.BlockMessage

	BestBlock  *blockchain.Block
	Blockchain *blockchain.Blockchain

	quit    chan struct{}
	solving bool
	target  *big.Int
}

func (miner *Miner) Start() {
	miner.quit = make(chan struct{})

	if miner.BestBlock != nil {
		miner.generateBlock()
		go miner.solve()
	}

	log.Info("Miner is running.")
}

func (miner *Miner) generateBlock() {
	miner.Lock.Lock()

	miner.block = network.NewBlockMessage()

	miner.block.Header = &network.BlockHeader{
		Version:       0,
		Flags:         []string{"olala", "olali"},
		HashPrevBlock: miner.BestBlock.Hash(),
		Height:        miner.BestBlock.Msg.Header.Height + 1,
		Nonce:         0,
		Timestamp:     time.Now(),
	}

	prevBlock, err := miner.Blockchain.Index.FindBlock(miner.block.Header.HashPrevBlock)
	if err != nil {
		log.WithError(err).Error("error finding the previous block in the miner")
		return
	}

	miner.block.Header.Bits, err = miner.Blockchain.CalcNextBlockDifficulty(prevBlock, blockchain.NewBlockFromBlockMessage(miner.block))
	if err != nil {
		log.WithError(err).Error("error computing the next block difficulty in the miner")
	}

	miner.target = blockchain.BitsToBig(miner.block.Header.Bits)

	miner.block.Txs = append(miner.block.Txs, newCoinbaseTx(blockchain.NewBlockFromBlockMessage(miner.block).CalcBlockSubsidy()))

	miner.block.Header.HashMerkleRoot = blockchain.ComputeMerkleRoot([]*utils.Hash{miner.block.Txs[0].Hash()})

	miner.Lock.Unlock()
}

func (miner *Miner) solve() {
	miner.solving = true

	log.WithFields(log.Fields{
		"target": fmt.Sprintf("%x", miner.target),
	}).Info("Miner is solving.")

	for miner.block.Header.Nonce < math.MaxUint64 {
		select {
		case <-miner.quit:
			return
		default:
			miner.Lock.Lock()
			miner.block.Header.Nonce += 1

			hash := miner.block.Header.Hash()

			if blockchain.HashToBig(hash).Cmp(miner.target) <= 0 {
				log.Info("Miner found a valid hash.")

				miner.solving = false
				log.Info("Miner is no more solving.")

				miner.Lock.Unlock()
				miner.Config.ProcessBlock(miner.block)

				return
			}
			miner.Lock.Unlock()
		}
	}

	miner.solving = false
	log.Info("Miner is no more solving.")

}

func (miner *Miner) UpdatePrevBlock(block *blockchain.Block) {
	miner.Lock.Lock()
	miner.BestBlock = block
	miner.target = blockchain.BitsToBig(block.Msg.Header.Bits)
	miner.Lock.Unlock()

	miner.generateBlock()

	if !miner.solving {
		go miner.solve()
	}
}

func (miner *Miner) Stop() {
	miner.quit <- struct{}{}

	log.Info("Miner is no more running.")
}
