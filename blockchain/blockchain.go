package blockchain

import (
	"bytes"
	"encoding/binary"
	"github.com/EnsicoinDevs/eccd/consensus"
	"github.com/EnsicoinDevs/eccd/network"
	"github.com/EnsicoinDevs/eccd/scripts"
	"github.com/EnsicoinDevs/eccd/utils"
	bolt "github.com/etcd-io/bbolt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"math/big"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type ValidationError struct {
	Reason string
}

func (err *ValidationError) Error() string {
	return "block invalid: " + err.Reason
}

type Config struct {
	DataDir       string
	OnPushedBlock func(*Block) error
	OnPoppedBlock func(*Block) error
}

type BlockchainStats struct {
	BestBlockHash   utils.Hash
	BestBlockHeight uint32
}

type Blockchain struct {
	config Config

	db *bolt.DB

	GenesisBlock *Block

	lock *sync.RWMutex

	stats *BlockchainStats

	orphans           map[utils.Hash]*Block
	prevHashToOrphans map[utils.Hash][]utils.Hash
	cache             map[utils.Hash]*network.BlockHeader
}

func NewBlockchain(config *Config) *Blockchain {
	return &Blockchain{
		config: *config,

		GenesisBlock: &genesisBlock,

		lock: &sync.RWMutex{},

		stats: &BlockchainStats{},

		orphans:           make(map[utils.Hash]*Block),
		prevHashToOrphans: make(map[utils.Hash][]utils.Hash),
		cache:             make(map[utils.Hash]*network.BlockHeader),
	}
}

var (
	blocksBucket    = []byte("blocks")
	heightsBucket   = []byte("heights")
	statsBucket     = []byte("stats")
	followingBucket = []byte("following")
	utxosBucket     = []byte("utxos")
	stxojBucket     = []byte("stxoj")
	workBucket      = []byte("work")
	ancestorBucket  = []byte("ancestor")
)

func (blockchain *Blockchain) Load() error {
	var err error

	blockchain.db, err = bolt.Open(filepath.Join(blockchain.config.DataDir, "data.db"), 0600, nil)
	if err != nil {
		return errors.Wrap(err, "error opening the blockchain database")
	}

	shouldStoreGenesisBlock := false

	blockchain.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(blocksBucket)
		if err != nil {
			return errors.Wrap(err, "error creating the blocks bucket")
		}

		genesisBlockInDb := b.Get(genesisBlock.Hash().Bytes())
		if genesisBlockInDb == nil {
			shouldStoreGenesisBlock = true
		}

		b, err = tx.CreateBucketIfNotExists(statsBucket)
		if err != nil {
			return errors.Wrap(err, "error creating the stats bucket")
		}

		bestBlockHashInDb := b.Get([]byte("bestBlockHash"))
		if bestBlockHashInDb == nil {
			b.Put([]byte("bestBlockHash"), genesisBlock.Hash().Bytes())
			b.Put([]byte("bestBlockHeight"), []byte{0, 0, 0, 0})
		} else {
			blockchain.stats.BestBlockHash = *utils.NewHash(bestBlockHashInDb)
			blockchain.stats.BestBlockHeight = binary.BigEndian.Uint32(b.Get([]byte("bestBlockHeight")))
		}

		_, err = tx.CreateBucketIfNotExists(heightsBucket)
		if err != nil {
			return errors.Wrap(err, "error creating the heights bucket")
		}

		_, err = tx.CreateBucketIfNotExists(followingBucket)
		if err != nil {
			return errors.Wrap(err, "error creating the following bucket")
		}

		_, err = tx.CreateBucketIfNotExists(utxosBucket)
		if err != nil {
			return errors.Wrap(err, "error creating the utxos bucket")
		}

		_, err = tx.CreateBucketIfNotExists(stxojBucket)
		if err != nil {
			return errors.Wrap(err, "error creating the stxoj bucket")
		}

		_, err = tx.CreateBucketIfNotExists(workBucket)
		if err != nil {
			return errors.Wrap(err, "error creating the work bucket")
		}

		_, err = tx.CreateBucketIfNotExists(ancestorBucket)
		if err != nil {
			return errors.Wrap(err, "error creating the ancestor bucket")
		}

		return nil
	})

	if shouldStoreGenesisBlock {
		blockchain.storeBlock(&genesisBlock)
	}

	return nil
}

func (blockchain *Blockchain) Stop() error {
	log.Debug("blockchain shutting down")
	defer log.Debug("blockchain shutdown complete")

	return blockchain.db.Close()
}

func (blockchain *Blockchain) GenerateBlockLocator() ([]*utils.Hash, error) {
	bestBlock, err := blockchain.FindBestBlock()
	if err != nil {
		return nil, err
	}

	return []*utils.Hash{
		bestBlock.Hash(),
		genesisBlock.Hash(),
	}, nil
}

func (blockchain *Blockchain) fetchUtxos(tx *Tx) (*Utxos, []*network.Outpoint, error) {
	utxos := newUtxos()
	var missings []*network.Outpoint

	err := blockchain.db.View(func(btx *bolt.Tx) error {
		b := btx.Bucket(utxosBucket)

		for _, input := range tx.Msg.Inputs {
			outpoint := input.PreviousOutput
			outpointBytes, err := outpoint.MarshalBinary()
			if err != nil {
				return errors.Wrap(err, "error marshaling an outpoint")
			}

			utxoBytes := b.Get(outpointBytes)
			if utxoBytes == nil {
				missings = append(missings, outpoint)
				continue
			}

			utxo := new(UtxoEntry)
			err = utxo.UnmarshalBinary(utxoBytes)
			if err != nil {
				return err
			}

			utxos.AddEntry(outpoint, utxo)
		}

		return nil
	})

	return utxos, missings, err
}

func (blockchain *Blockchain) FetchUtxos(tx *Tx) (*Utxos, []*network.Outpoint, error) {
	log.WithFields(log.Fields{
		"func":  "FetchUtxos",
		"mutex": "blockchain",
	}).Trace("rlocking")
	blockchain.lock.RLock()
	defer blockchain.lock.RUnlock()
	defer log.WithFields(log.Fields{
		"func":  "FetchUtxos",
		"mutex": "blockchain",
	}).Trace("runlocking")

	return blockchain.fetchUtxos(tx)
}

func (blockchain *Blockchain) fetchAllUtxos() (*Utxos, error) {
	utxos := newUtxos()

	err := blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(utxosBucket)

		b.ForEach(func(outpointBytes, utxoBytes []byte) error {
			outpoint := new(network.Outpoint)
			err := outpoint.UnmarshalBinary(outpointBytes)
			if err != nil {
				return err
			}

			utxo := new(UtxoEntry)
			err = utxo.UnmarshalBinary(utxoBytes)
			if err != nil {
				return err
			}

			utxos.AddEntry(outpoint, utxo)

			return nil
		})

		return nil
	})

	return utxos, err
}

func (blockchain *Blockchain) FetchAllUtxos() (*Utxos, error) {
	log.WithFields(log.Fields{
		"func":  "FetchAllUtxos",
		"mutex": "blockchain",
	}).Trace("rlocking")
	blockchain.lock.RLock()
	defer blockchain.lock.RUnlock()
	defer log.WithFields(log.Fields{
		"func":  "FetchAllUtxos",
		"mutex": "blockchain",
	}).Trace("runlocking")

	return blockchain.fetchAllUtxos()
}

func (blockchain *Blockchain) storeBlock(block *Block) error {
	if block.Hash().IsEqual(genesisBlock.Hash()) {
		if err := blockchain.storeWork(block.Hash(), big.NewInt(0)); err != nil {
			return err
		}
	} else {
		prevBlockWork, err := blockchain.getWork(block.Msg.Header.HashPrevBlock)
		if err != nil {
			return err
		}

		err = blockchain.storeWork(block.Hash(), prevBlockWork.Add(prevBlockWork, block.GetWork()))
		if err != nil {
			return err
		}

		if block.Msg.Header.Height == consensus.BLOCKS_PER_RETARGET {
			if err = blockchain.storeAncestor(block.Hash(), genesisBlock.Hash()); err != nil {
				return err
			}
		} else if block.Msg.Header.Height > consensus.BLOCKS_PER_RETARGET {
			prevBlockAncestorHash, err := blockchain.GetAncestor(block.Msg.Header.HashPrevBlock)
			if err != nil {
				return err
			}

			prevBlockAncestorFollowingHash, err := blockchain.GetFollowing(prevBlockAncestorHash)
			if err != nil {
				return err
			}

			err = blockchain.storeAncestor(block.Hash(), prevBlockAncestorFollowingHash)
			if err != nil {
				return err
			}
		}
	}

	return blockchain.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(blocksBucket)

		blockBytes, err := block.MarshalBinary()
		if err != nil {
			return err
		}

		return b.Put(block.Hash().Bytes(), blockBytes)
	})
}

func (blockchain *Blockchain) FindBlockByHash(hash *utils.Hash) (*Block, error) {
	var block *Block

	err := blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(blocksBucket)

		blockBytes := b.Get(hash.Bytes())
		if blockBytes == nil {
			return nil
		}

		block = NewBlock()

		block.UnmarshalBinary(blockBytes)

		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "error finding a block")
	}

	return block, nil
}

func (blockchain *Blockchain) HaveBlock(hash *utils.Hash) (bool, error) {
	var exists bool

	err := blockchain.db.View(func(tx *bolt.Tx) error {
		exists = tx.Bucket(blocksBucket).Get(hash.Bytes()) != nil

		return nil
	})
	if err != nil {
		return false, errors.Wrap(err, "error finding a block")
	}

	return exists, nil
}

func (blockchain *Blockchain) FindBestBlock() (*Block, error) {
	var bestBlockHash *utils.Hash

	err := blockchain.db.View(func(tx *bolt.Tx) error {
		bestBlockHash = utils.NewHash(tx.Bucket(statsBucket).Get([]byte("bestBlockHash")))

		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "error finding the best block hash")
	}

	block, err := blockchain.FindBlockByHash(bestBlockHash)
	if err != nil {
		return nil, errors.Wrap(err, "error finding the best block")
	}

	return block, nil
}

func (blockchain *Blockchain) findBlockHashesStartingAt(hash *utils.Hash, limit int) ([]*utils.Hash, error) {
	var hashes []*utils.Hash

	err := blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(followingBucket)

		for limit > 0 {
			hashBytes := b.Get(hash.Bytes())
			if hashBytes == nil {
				break
			}

			hash = utils.NewHash(hashBytes)

			hashes = append(hashes, hash)
			limit--
		}

		return nil
	})
	if err != nil {
		return nil, errors.Wrap(nil, "error finding all the hashes")
	}

	return hashes, nil
}

func (blockchain *Blockchain) FindBlockHashesStartingAt(hash *utils.Hash, limit int) ([]*utils.Hash, error) {
	log.WithFields(log.Fields{
		"func":  "FindBlockHashesStartingAt",
		"mutex": "blockchain",
	}).Trace("rlocking")
	blockchain.lock.RLock()
	defer blockchain.lock.RUnlock()
	defer log.WithFields(log.Fields{
		"func":  "FindBlockHashesStartingAt",
		"mutex": "blockchain",
	}).Trace("runlocking")

	return blockchain.findBlockHashesStartingAt(hash, limit)
}

func (blockchain *Blockchain) CalcNextBlockDifficulty(timestamp time.Time, ancestorTimestamp time.Time, ancestorTarget *big.Int) (*big.Int, error) {
	realizedTimespan := uint64(timestamp.Unix() - ancestorTimestamp.Unix())
	if realizedTimespan < consensus.MIN_RETARGET_TIMESPAN {
		realizedTimespan = consensus.MIN_RETARGET_TIMESPAN
	} else if realizedTimespan > consensus.MAX_RETARGET_TIMESPAN {
		realizedTimespan = consensus.MAX_RETARGET_TIMESPAN
	}

	target := new(big.Int).Mul(ancestorTarget, big.NewInt(int64(realizedTimespan)))
	target.Div(target, big.NewInt(consensus.BLOCKS_MEAN_TIMESPAN*consensus.BLOCKS_PER_RETARGET))

	return target, nil
}

func (blockchain *Blockchain) validateBlock(block *Block) (error, error) {
	parentBlock, err := blockchain.FindBlockByHash(block.Msg.Header.HashPrevBlock)
	if err == nil {
		return nil, err
	}
	if parentBlock.Msg.Header.Height+1 != block.Msg.Header.Height {
		return errors.New("height is not equal to the previous block height + 1"), nil
	}

	nextTarget := parentBlock.Msg.Header.Target
	if block.Msg.Header.Height >= consensus.BLOCKS_PER_RETARGET {
		ancestorHash, err := blockchain.GetAncestor(block.Hash())
		if err != nil {
			return nil, err
		}

		ancestor, err := blockchain.FindBlockByHash(ancestorHash)
		if err != nil {
			return nil, err
		}

		nextTarget, err = blockchain.CalcNextBlockDifficulty(block.Msg.Header.Timestamp, ancestor.Msg.Header.Timestamp, ancestor.Msg.Header.Target)
		if err != nil {
			return nil, err
		}
	}

	if block.Msg.Header.Target.Cmp(nextTarget) != 0 {
		return errors.New("the difficulty is invalid"), nil
	}

	if block.Txs[0].Msg.Flags[0] != strconv.Itoa(int(block.Msg.Header.Height)) {
		return errors.New("the first flag of the coinbase is not the block height"), nil
	}

	var totalFees uint64

	for _, tx := range block.Txs[1:] { // TODO: huuuum
		utxos, notfound, err := blockchain.fetchUtxos(tx)
		if err != nil {
			return nil, err
		}
		if len(notfound) != 0 {
			return errors.New("a tx try to spend an spent / unknown tx"), nil
		}

		valid, fees, err := blockchain.validateTx(tx, block, utxos)
		if err == nil {
			return nil, nil
		}
		if !valid {
			return errors.New("a tx is invalid"), nil
		}

		totalFees += fees
	}

	var totalCoinbaseAmount uint64

	for _, output := range block.Txs[0].Msg.Outputs {
		totalCoinbaseAmount += output.Value
	}

	if totalCoinbaseAmount != totalFees+block.CalcBlockSubsidy() {
		return errors.New("coinbase value is invalid"), nil
	}

	return nil, nil
}

func (blockchain *Blockchain) ValidateBlock(block *Block) (error, error) {
	log.WithFields(log.Fields{
		"func":  "ValidateBlock",
		"mutex": "blockchain",
	}).Trace("rlocking")
	blockchain.lock.RLock()
	defer blockchain.lock.RUnlock()
	defer log.WithFields(log.Fields{
		"func":  "ValidateBlock",
		"mutex": "blockchain",
	}).Trace("runlocking")

	return blockchain.validateBlock(block)
}

func (blockchain *Blockchain) validateTx(tx *Tx, block *Block, utxos *Utxos) (bool, uint64, error) {
	var inputsSum uint64

	for _, input := range tx.Msg.Inputs {
		utxo := utxos.FindEntry(input.PreviousOutput)

		if utxo.IsCoinBase() {
			if block.Msg.Header.Height-utxo.blockHeight < consensus.COINBASE_MATURITY {
				return false, 0, nil
			}
		}

		inputsSum += utxo.Amount()
	}

	var outputsSum uint64

	for _, output := range tx.Msg.Outputs {
		outputsSum += output.Value
	}

	if inputsSum <= outputsSum {
		return false, 0, nil
	}

	for _, input := range tx.Msg.Inputs {
		utxo := utxos.FindEntry(input.PreviousOutput)
		script := scripts.NewScript(tx.Msg, input, utxo.Amount(), utxo.Script(), input.Script)
		valid, err := script.Validate()
		if err != nil {
			return false, 0, err
		}
		if !valid {
			return false, 0, nil
		}
	}

	return false, inputsSum - outputsSum, nil
}

func (blockchain *Blockchain) Validatetx(tx *Tx, block *Block, utxos *Utxos) (bool, uint64, error) {
	log.WithFields(log.Fields{
		"func":  "ValidateTx",
		"mutex": "blockchain",
	}).Trace("rlocking")
	blockchain.lock.RLock()
	defer blockchain.lock.RUnlock()
	defer log.WithFields(log.Fields{
		"func":  "ValidateTx",
		"mutex": "blockchain",
	}).Trace("runlocking")

	return blockchain.Validatetx(tx, block, utxos)
}

func (blockchain *Blockchain) storeWork(blockHash *utils.Hash, work *big.Int) error {
	return blockchain.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(workBucket)

		return b.Put(blockHash.Bytes(), work.Bytes())
	})
}

func (blockchain *Blockchain) getWork(blockHash *utils.Hash) (*big.Int, error) {
	var work = new(big.Int)

	return work, blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(workBucket)

		workBytes := b.Get(blockHash.Bytes())
		if workBytes == nil {
			return errors.New("block not found")
		}

		work.SetBytes(workBytes)

		return nil
	})
}

func (blockchain *Blockchain) storeAncestor(blockHash *utils.Hash, ancestorHash *utils.Hash) error {
	return blockchain.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(ancestorBucket)

		return b.Put(blockHash.Bytes(), ancestorHash.Bytes())
	})
}

func (blockchain *Blockchain) GetAncestor(blockHash *utils.Hash) (*utils.Hash, error) {
	var ancestorHash *utils.Hash

	if err := blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(ancestorBucket)

		ancestorHashBytes := b.Get(blockHash.Bytes())
		if ancestorHashBytes == nil {
			return errors.New("ancestor block not found")
		}

		ancestorHash = utils.NewHash(ancestorHashBytes)

		return nil
	}); err != nil {
		return nil, err
	}

	return ancestorHash, nil
}

func (blockchain *Blockchain) GetFollowing(blockHash *utils.Hash) (*utils.Hash, error) {
	var followingHash *utils.Hash

	if err := blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(followingBucket)

		followingHashBytes := b.Get(blockHash.Bytes())
		if followingHashBytes == nil {
			return errors.New("following block not found")
		}

		followingHash = utils.NewHash(followingHashBytes)

		return nil
	}); err != nil {
		return nil, err
	}

	return followingHash, nil
}

func (blockchain *Blockchain) fetchStxojEntry(blockHash *utils.Hash) ([]*UtxoEntry, error) {
	var utxoEntries []*UtxoEntry

	err := blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(stxojBucket)

		buf := bytes.NewBuffer(b.Get(blockHash.Bytes()))

		for buf.Len() != 0 {
			utxoEntry, err := ReadUtxoEntry(buf)
			if err != nil {
				return err
			}

			utxoEntries = append(utxoEntries, utxoEntry)
		}

		return nil
	})

	return utxoEntries, err
}

func (blockchain *Blockchain) FetchStxojEntry(blockHash *utils.Hash) ([]*UtxoEntry, error) {
	log.WithFields(log.Fields{
		"func":  "FetchStxojEntry",
		"mutex": "blockchain",
	}).Trace("rlocking")
	blockchain.lock.RLock()
	defer blockchain.lock.RUnlock()
	defer log.WithFields(log.Fields{
		"func":  "FetchStxojEntry",
		"mutex": "blockchain",
	}).Trace("runlocking")

	return blockchain.fetchStxojEntry(blockHash)
}

func (blockchain *Blockchain) getBlockHashAtHeight(height uint32) (*utils.Hash, error) {
	var hash *utils.Hash

	err := blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(heightsBucket)

		heightBytes := make([]byte, 4)

		binary.BigEndian.PutUint32(heightBytes, height)

		hash = utils.NewHash(b.Get(heightBytes))

		return nil
	})

	return hash, err
}

func (blockchain *Blockchain) GetBlockHashAtHeight(height uint32) (*utils.Hash, error) {
	log.WithFields(log.Fields{
		"func":  "GetBlockHashAtHeight",
		"mutex": "blockchain",
	}).Trace("rlocking")
	blockchain.lock.RLock()
	defer blockchain.lock.RUnlock()
	defer log.WithFields(log.Fields{
		"func":  "GetBlockHashAtHeight",
		"mutex": "blockchain",
	}).Trace("runlocking")

	return blockchain.getBlockHashAtHeight(height)
}

func (blockchain *Blockchain) pushBlock(block *Block) error {
	longestChain, err := blockchain.FindBestBlock()
	if err != nil {
		return err
	}

	if !longestChain.Hash().IsEqual(block.Msg.Header.HashPrevBlock) {
		return errors.New("block must extend the best chain")
	}

	utxos, err := blockchain.fetchAllUtxos()
	if err != nil {
		return err
	}

	var stxoj []*UtxoEntry

	for _, tx := range block.Txs {
		for _, input := range tx.Msg.Inputs {
			removedEntry := utxos.RemoveEntry(input.PreviousOutput)
			stxoj = append(stxoj, removedEntry)
		}

		for index, output := range tx.Msg.Outputs {
			utxos.AddEntry(&network.Outpoint{
				Hash:  *tx.Hash(),
				Index: uint32(index),
			}, &UtxoEntry{
				amount:      output.Value,
				script:      output.Script,
				blockHeight: block.Msg.Header.Height,
				coinBase:    tx.IsCoinBase(),
			})
		}
	}

	if err := blockchain.db.Update(func(tx *bolt.Tx) error {
		var err error

		if err = dbStoreBestBlock(tx, block.Hash()); err != nil {
			return err
		}

		if err = dbStoreStxojEntry(tx, block.Hash(), stxoj); err != nil {
			return err
		}

		if err = dbStoreUtxos(tx, utxos); err != nil {
			return err
		}

		if err = dbStoreBlockHashAtHeight(tx, block.Hash(), block.Msg.Header.Height); err != nil {
			return err
		}

		if err = dbStoreFollowing(tx, block.Msg.Header.HashPrevBlock, block.Hash()); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	if blockchain.config.OnPushedBlock != nil {
		blockchain.config.OnPushedBlock(block)
	}

	return nil
}

func (blockchain *Blockchain) popBlock(block *Block) error {
	longestChain, err := blockchain.FindBestBlock()
	if err != nil {
		return err
	}

	if !longestChain.Hash().IsEqual(block.Hash()) {
		return errors.New("block must be the best block")
	}

	utxos, err := blockchain.fetchAllUtxos()
	if err != nil {
		return err
	}

	stxoj, err := blockchain.fetchStxojEntry(block.Hash())
	if err != nil {
		return err
	}

	var currentStxoId uint64

	for _, tx := range block.Txs {
		for index := range tx.Msg.Outputs {
			utxos.RemoveEntry(&network.Outpoint{
				Hash:  *tx.Hash(),
				Index: uint32(index),
			})
		}

		for _, input := range tx.Msg.Inputs {
			stxo := stxoj[currentStxoId]
			currentStxoId++

			entry := utxos.FindEntry(input.PreviousOutput)
			if entry == nil {
				entry = newUtxoEntry()
				utxos.AddEntry(input.PreviousOutput, entry)
			}

			entry.amount = stxo.amount
			entry.script = stxo.script
			entry.blockHeight = stxo.blockHeight
			entry.coinBase = stxo.coinBase
		}
	}

	if err = blockchain.db.Update(func(tx *bolt.Tx) error {
		var err error

		if err = dbStoreUtxos(tx, utxos); err != nil {
			return err
		}

		if err = dbRemoveStxojEntry(tx, block.Hash()); err != nil {
			return err
		}

		if err = dbRemoveFollowing(tx, block.Msg.Header.HashPrevBlock); err != nil {
			return err
		}

		return dbStoreBestBlock(tx, block.Msg.Header.HashPrevBlock)
	}); err != nil {
		return err
	}

	blockchain.config.OnPoppedBlock(block)

	return nil
}

func (blockchain *Blockchain) findForkingBlock(blockA *Block, blockB *Block) (*Block, error) {
	var err error

	// blockA must be the higher block
	for blockB.Msg.Header.Height > blockA.Msg.Header.Height {
		blockA, blockB = blockB, blockA
	}

	// If blockA height is superior to blockB height, we walk back to the blockB height
	for blockA.Msg.Header.Height > blockB.Msg.Header.Height {
		blockA, err = blockchain.FindBlockByHash(blockA.Msg.Header.HashPrevBlock)
		if err != nil {
			return nil, err
		}
	}

	// While the two blocks are different, we walk back of one block
	for blockA != nil && !blockB.Hash().IsEqual(blockA.Hash()) {
		blockA, err = blockchain.FindBlockByHash(blockA.Msg.Header.HashPrevBlock)
		if err != nil {
			return nil, err
		}

		blockB, err = blockchain.FindBlockByHash(blockB.Msg.Header.HashPrevBlock)
		if err != nil {
			return nil, err
		}
	}

	return blockA, nil
}

func (blockchain *Blockchain) findBlockHashesBetween(startBlockHash, endBlockHash *utils.Hash) ([]*utils.Hash, error) {
	current := endBlockHash
	blockHashes := []*utils.Hash{current}

	for !current.IsEqual(startBlockHash) && !current.IsEqual(genesisBlock.Hash()) {
		currentBlock, err := blockchain.FindBlockByHash(current)
		if err != nil {
			return nil, err
		}

		current = currentBlock.Msg.Header.HashPrevBlock
		blockHashes = append(blockHashes, current)
	}

	// Reverse the slice
	for i, j := 0, len(blockHashes)-1; i < j; i, j = i+1, j-1 {
		blockHashes[i], blockHashes[j] = blockHashes[j], blockHashes[i]
	}

	return blockHashes, nil
}

func (blockchain *Blockchain) acceptBlock(block *Block) (bool, error) {
	err := blockchain.storeBlock(block)
	if err != nil {
		return false, err
	}

	bestBlock, err := blockchain.FindBestBlock()
	if err != nil {
		return false, err
	}

	if bestBlock.Hash().IsEqual(block.Msg.Header.HashPrevBlock) {
		if err = blockchain.pushBlock(block); err != nil {
			return false, err
		}

		return true, nil
	}

	bestBlockWork, err := blockchain.getWork(bestBlock.Hash())
	if err != nil {
		return false, err
	}

	blockWork, err := blockchain.getWork(block.Hash())
	if err != nil {
		return false, err
	}

	if bestBlockWork.Cmp(blockWork) > 0 {
		return true, nil
	}

	forkingBlock, err := blockchain.findForkingBlock(bestBlock, block)
	if err != nil {
		return false, err
	}
	if forkingBlock == nil {
		return false, errors.New("can't find the forking block")
	}

	current := bestBlock

	for !current.Hash().IsEqual(forkingBlock.Hash()) {
		if err = blockchain.popBlock(current); err != nil {
			return false, err
		}

		current, err = blockchain.FindBestBlock()
		if err != nil {
			return false, err
		}
	}

	blockHashesToPush, err := blockchain.findBlockHashesBetween(forkingBlock.Hash(), block.Hash())
	if err != nil {
		return false, err
	}

	if len(blockHashesToPush) < 1 {
		return false, errors.New("nani the fuck")
	}

	blockHashesToPush = blockHashesToPush[1:]

	for _, blockHashToPush := range blockHashesToPush {
		blockToPush, err := blockchain.FindBlockByHash(blockHashToPush)
		if err != nil {
			return false, err
		}

		if err = blockchain.pushBlock(blockToPush); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (blockchain *Blockchain) addOrphan(block *Block) {
	blockchain.orphans[*block.Hash()] = block
	blockchain.prevHashToOrphans[*block.Msg.Header.HashPrevBlock] = append(blockchain.prevHashToOrphans[*block.Msg.Header.HashPrevBlock], *block.Hash())
}

func (blockchain *Blockchain) processOrphans(acceptedBlock *Block) error {
	for _, blockHash := range blockchain.prevHashToOrphans[*acceptedBlock.Hash()] {
		block, err := blockchain.findOrphan(&blockHash)
		if err != nil {
			return err
		}

		valid, err := blockchain.validateBlock(block)
		if err != nil {
			return err
		}
		if valid != nil {
			blockchain.removeOrphan(block.Hash())
		}

		_, err = blockchain.acceptBlock(block)
		if err != nil {
			return err
		}

		err = blockchain.processOrphans(block)
		if err != nil {
			return err
		}
	}

	return nil
}

func (blockchain *Blockchain) findOrphan(hash *utils.Hash) (*Block, error) {
	return blockchain.orphans[*hash], nil
}

func (blockchain *Blockchain) FindOrphan(hash *utils.Hash) (*Block, error) {
	log.WithFields(log.Fields{
		"func":  "FindOrphan",
		"mutex": "blockchain",
	}).Trace("rlocking")
	blockchain.lock.RLock()
	defer blockchain.lock.RUnlock()
	defer log.WithFields(log.Fields{
		"func":  "FindOrphan",
		"mutex": "blockchain",
	}).Trace("runlocking")

	return blockchain.findOrphan(hash)
}

func (blockchain *Blockchain) removeOrphan(hash *utils.Hash) {
	for _, blockHash := range blockchain.prevHashToOrphans[*hash] {
		blockchain.removeOrphan(&blockHash)
	}

	delete(blockchain.prevHashToOrphans, *hash)
	delete(blockchain.orphans, *hash)
}

func (blockchain *Blockchain) ProcessBlock(block *Block) (error, error) {
	log.WithFields(log.Fields{
		"func":  "ProcessBlock",
		"mutex": "blockchain",
	}).Trace("locking")
	blockchain.lock.Lock()
	defer blockchain.lock.Unlock()
	defer log.WithFields(log.Fields{
		"func":  "ProcessBlock",
		"mutex": "blockchain",
	}).Trace("unlocking")

	exists, err := blockchain.HaveBlock(block.Hash())
	if err != nil {
		return nil, err
	}
	if exists {
		return &ValidationError{"a block with the same hash exist"}, nil
	}

	orphanBlock, err := blockchain.findOrphan(block.Hash())
	if err != nil {
		return nil, nil
	}
	if orphanBlock != nil {
		return &ValidationError{"an orphan block with the same hash exist"}, nil
	}

	if valid := block.IsSane(); valid != nil {
		return errors.Wrap(valid, "the block is not sane"), nil
	}

	exists, err = blockchain.HaveBlock(block.Msg.Header.HashPrevBlock)
	if err != nil {
		return nil, err
	}
	if !exists {
		log.WithFields(log.Fields{
			"hash":   block.Hash().String(),
			"height": block.Msg.Header.Height,
		}).Info("accepted orphan block")

		blockchain.addOrphan(block)

		return nil, nil
	}

	valid, err := blockchain.validateBlock(block)
	if err != nil {
		return nil, err
	}
	if valid != nil {
		return errors.Wrap(valid, "the block is invalid"), nil
	}

	_, err = blockchain.acceptBlock(block)
	if err != nil {
		return nil, err
	}

	err = blockchain.processOrphans(block)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
