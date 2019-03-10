package miner

import (
	"github.com/EnsicoinDevs/ensicoincoin/blockchain"
	"github.com/EnsicoinDevs/ensicoincoin/mempool"
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"github.com/EnsicoinDevs/ensicoincoin/utils"
	log "github.com/sirupsen/logrus"
	"math"
	"math/big"
	"strconv"
	"time"
)

type Config struct {
	ProcessBlock func(*network.BlockMessage)
}

type Miner struct {
	Config *Config

	block *network.BlockMessage

	BestBlock *blockchain.Block

	Blockchain *blockchain.Blockchain
	Mempool    *mempool.Mempool

	quit   chan struct{}
	target *big.Int
}

func (miner *Miner) Start() {
	miner.quit = make(chan struct{})

	log.Info("Miner is now running.")

	go miner.startMainLoop()
}

func (miner *Miner) startMainLoop() {
	for {
		select {
		case <-miner.quit:
			return
		default:
		}

		if miner.BestBlock == nil {
			log.Debug("bestblock is nil")
			time.Sleep(time.Second)
			continue
		}

		miner.generateBlock()

		blockFound := miner.solve()

		if blockFound {
			log.Debug("block found")

			if miner.block.Header.Height == miner.BestBlock.Msg.Header.Height+1 {
				miner.BestBlock = nil
				miner.Config.ProcessBlock(miner.block)
			} else {
				log.Debug("the miner found a nonce for an outdated block")
			}
		}
	}
}

func (miner *Miner) solve() bool {
	log.WithField("HashPrevBlock", miner.block.Header.HashPrevBlock).Debug("solver is running")

	var hash *utils.Hash
	var bestHeight uint32

	ticker := time.NewTicker(time.Second)

	for miner.block.Header.Nonce < math.MaxUint64 {
		select {
		case <-miner.quit:
			miner.quit <- struct{}{}
			return false
		case <-ticker.C:
			log.WithField("nonce", miner.block.Header.Nonce).Debug("solving...")

			bestHeight = miner.BestBlock.Msg.Header.Height

			if bestHeight+1 != miner.block.Header.Height {
				log.Debug("bestblock is no more the best block")

				return false
			}
		default:
		}

		miner.block.Header.Nonce += 1

		hash = miner.block.Header.Hash()

		if blockchain.HashToBig(hash).Cmp(miner.target) <= 0 {
			return true
		}
	}

	return false
}

func (miner *Miner) generateBlock() {
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

	miner.block.Txs[0].Flags[0] = strconv.Itoa(int(miner.block.Header.Height))

	for _, tx := range miner.Mempool.FetchTxs() {
		miner.block.Txs = append(miner.block.Txs, tx.Msg)
	}

	var hashes []*utils.Hash

	for _, tx := range miner.block.Txs {
		hashes = append(hashes, tx.Hash())
	}

	miner.block.Header.HashMerkleRoot = blockchain.ComputeMerkleRoot(hashes)
}

func (miner *Miner) UpdateBestBlock(block *blockchain.Block) {
	miner.BestBlock = block
}

func (miner *Miner) Stop() {
	miner.quit <- struct{}{}

	log.Info("Miner is no more running.")
}
