package blockchain

import (
	"bytes"
	"encoding/binary"
	"github.com/EnsicoinDevs/ensicoincoin/consensus"
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"github.com/EnsicoinDevs/ensicoincoin/scripts"
	"github.com/EnsicoinDevs/ensicoincoin/utils"
	bolt "github.com/etcd-io/bbolt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"math/big"
	"sync"
)

type Blockchain struct {
	db *bolt.DB

	GenesisBlock *Block

	Index *BlockIndex

	orphansLock       sync.Mutex
	orphans           map[*utils.Hash]*Block
	prevHashToOrphans map[*utils.Hash][]*utils.Hash
}

func NewBlockchain() *Blockchain {
	blockchain := &Blockchain{
		GenesisBlock: &genesisBlock,
	}

	blockchain.Index = NewBlockIndex(blockchain)

	return blockchain
}

var (
	blocksBucket    = []byte("blocks")
	heightsBucket   = []byte("heights")
	statsBucket     = []byte("stats")
	followingBucket = []byte("following")
	utxosBucket     = []byte("utxos")
	stxojBucket     = []byte("stxoj")
)

func (blockchain *Blockchain) Load() error {
	var err error
	blockchain.db, err = bolt.Open("blockchain.db", 0600, nil)
	if err != nil {
		return errors.Wrap(err, "error opening the blockchain database")
	}

	shouldStoreGenesisBlock := false

	blockchain.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(blocksBucket)
		if err != nil {
			return errors.Wrap(err, "error creating the blocks bucket")
		}

		genesisBlockInDb := b.Get(genesisBlock.Hash()[:])
		if genesisBlockInDb == nil {
			shouldStoreGenesisBlock = true
		}

		b, err = tx.CreateBucketIfNotExists(statsBucket)
		if err != nil {
			return errors.Wrap(err, "error creating the stats bucket")
		}

		longestChainInDb := b.Get([]byte("longestChain"))
		if longestChainInDb == nil {
			b.Put([]byte("longestChain"), genesisBlock.Hash()[:])
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

		return nil
	})

	if shouldStoreGenesisBlock {
		blockchain.StoreBlock(&genesisBlock)
	}

	return nil
}

func (blockchain *Blockchain) FetchUtxos(tx *Tx) (*Utxos, []*network.Outpoint, error) {
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

			var utxo *UtxoEntry
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

func (blockchain *Blockchain) FetchAllUtxos() (*Utxos, error) {
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

func (blockchain *Blockchain) StoreBlock(block *Block) error {
	err := blockchain.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(blocksBucket)

		blockBytes, err := block.MarshalBinary()
		if err != nil {
			return err
		}

		b.Put(block.Hash()[:], blockBytes)

		return nil
	})

	return err
}

func (blockchain *Blockchain) FindBlockByHash(hash *utils.Hash) (*Block, error) {
	var block *Block

	err := blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(blocksBucket)

		blockBytes := b.Get(hash[:])
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

func (blockchain *Blockchain) FindLongestChain() (*Block, error) {
	var longestChainHash *utils.Hash

	err := blockchain.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(statsBucket)
		longestChainHash = utils.NewHash(b.Get([]byte("longestChain")))

		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "error finding the longest chain hash")
	}

	block, err := blockchain.FindBlockByHash(longestChainHash)
	if err != nil {
		return nil, errors.Wrap(err, "error finding the longest chain")
	}

	return block, nil
}

func (blockchain *Blockchain) FindBlockHashesStartingAt(hash *utils.Hash) ([]*utils.Hash, error) {
	var hashes []*utils.Hash

	err := blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(followingBucket)

		c := b.Cursor()

		current := hash

		for k, v := c.Seek(hash[:]); k != nil; k, v = c.Seek(current[:]) {
			hash := utils.NewHash(v)
			hashes = append(hashes, hash)

			current = hash
		}

		return nil
	})
	if err != nil {
		return nil, errors.Wrap(nil, "error finding all the hashes")
	}

	return hashes, nil
}

func (blockchain *Blockchain) CalcNextBlockDifficulty(block *BlockIndexEntry, nextBlock *Block) (uint32, error) {
	if nextBlock.Msg.Header.Height < consensus.BLOCKS_PER_RETARGET {
		return block.bits, nil
	}

	lastRetargetBlockEntry, err := blockchain.Index.findAncestor(nextBlock, nextBlock.Msg.Header.Height-consensus.BLOCKS_PER_RETARGET)
	if err != nil {
		return 0, err
	}

	realizedTimespan := uint64(nextBlock.Msg.Header.Timestamp.Unix()) - uint64(lastRetargetBlockEntry.timestamp.Unix())
	if realizedTimespan < consensus.MIN_RETARGET_TIMESPAN {
		realizedTimespan = consensus.MIN_RETARGET_TIMESPAN
	} else if realizedTimespan > consensus.MAX_RETARGET_TIMESPAN {
		realizedTimespan = consensus.MAX_RETARGET_TIMESPAN
	}

	targetBefore := BitsToBig(block.bits)
	target := new(big.Int).Mul(targetBefore, big.NewInt(int64(realizedTimespan)))
	target.Div(target, big.NewInt(consensus.BLOCKS_MEAN_TIMESPAN*consensus.BLOCKS_PER_RETARGET))

	return BigToBits(target), nil
}

func (blockchain *Blockchain) ValidateBlock(block *Block) (error, error) {
	parentBlock, err := blockchain.Index.FindBlock(block.Msg.Header.HashPrevBlock)
	if parentBlock.height+1 != block.Msg.Header.Height {
		return errors.New("height is not equal to the previous block height + 1"), nil
	}

	nextBits, err := blockchain.CalcNextBlockDifficulty(parentBlock, block)
	if err != nil {
		return nil, err
	}
	if block.Msg.Header.Bits != nextBits {
		return errors.New("the difficulty is invalid"), nil
	}

	var totalFees uint64

	for _, tx := range block.Txs[1:] { // TODO: huuuum
		utxos, notfound, err := blockchain.FetchUtxos(tx)
		if err != nil {
			return nil, err
		}
		if len(notfound) != 0 {
			return errors.New("a tx try to spend an spent / unknown tx"), nil
		}

		valid, fees, err := blockchain.ValidateTx(tx, block, utxos)
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

func (blockchain *Blockchain) ValidateTx(tx *Tx, block *Block, utxos *Utxos) (bool, uint64, error) {
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

func (blockchain *Blockchain) StoreUtxos(utxos *Utxos) error {
	return blockchain.db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket(utxosBucket)
		if err != nil {
			return err
		}

		b, err := tx.CreateBucket(utxosBucket)
		if err != nil {
			return err
		}

		for outpoint, entry := range utxos.entries {
			outpointBytes, err := outpoint.MarshalBinary()
			if err != nil {
				return err
			}

			entryBytes, err := entry.MarshalBinary()
			if err != nil {
				return err
			}

			err = b.Put(outpointBytes, entryBytes)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (blockchain *Blockchain) StoreFollowing(block *utils.Hash, nextBlock *utils.Hash) error {
	return blockchain.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(followingBucket)

		return b.Put(block[:], nextBlock[:])
	})
}

func (blockchain *Blockchain) RemoveFollowing(block *utils.Hash) error {
	return blockchain.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(followingBucket)

		return b.Delete(block[:])
	})
}

func (blockchain *Blockchain) RemoveStxojEntry(blockHash *utils.Hash) error {
	return blockchain.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(stxojBucket)

		return b.Delete(blockHash[:])
	})
}

func (blockchain *Blockchain) StoreStxojEntry(blockHash *utils.Hash, entry []*UtxoEntry) error {
	return blockchain.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(stxojBucket)

		buf := bytes.NewBuffer(nil)

		for _, stxo := range entry {
			err := WriteUtxoEntry(buf, stxo)
			if err != nil {
				return err
			}
		}

		return b.Put(blockHash[:], buf.Bytes())
	})
}

func (blockchain *Blockchain) FetchStxojEntry(blockHash *utils.Hash) ([]*UtxoEntry, error) {
	var utxoEntries []*UtxoEntry

	err := blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(stxojBucket)

		buf := bytes.NewBuffer(b.Get(blockHash[:]))

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

func (blockchain *Blockchain) StoreLongestChain(blockHash *utils.Hash) error {
	return blockchain.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(statsBucket)

		return b.Put([]byte("longestChain"), blockHash[:])
	})
}

func (blockchain *Blockchain) StoreBlockHashAtHeight(blockHash *utils.Hash, height uint32) error {
	return blockchain.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(heightsBucket)

		heightBytes := make([]byte, 4)

		binary.BigEndian.PutUint32(heightBytes, height)

		return b.Put(heightBytes, blockHash[:])
	})
}

func (blockchain *Blockchain) GetBlockHashAtHeight(height uint32) (*utils.Hash, error) {
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

func (blockchain *Blockchain) PushBlock(block *Block) error {
	longestChain, err := blockchain.FindLongestChain()
	if err != nil {
		return err
	}

	if !longestChain.Hash().IsEqual(block.Msg.Header.HashPrevBlock) {
		return errors.New("block must extend the best chain")
	}

	utxos, err := blockchain.FetchAllUtxos()
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
				Hash:  tx.Hash(),
				Index: uint32(index),
			}, &UtxoEntry{
				amount:      output.Value,
				script:      output.Script,
				blockHeight: block.Msg.Header.Height,
				coinBase:    tx.IsCoinBase(),
			})
		}
	}

	err = blockchain.StoreLongestChain(block.Hash())
	if err != nil {
		return err
	}

	err = blockchain.StoreFollowing(longestChain.Hash(), block.Hash())
	if err != nil {
		return err
	}

	err = blockchain.StoreStxojEntry(block.Hash(), stxoj)
	if err != nil {
		return err
	}

	err = blockchain.StoreUtxos(utxos)
	if err != nil {
		return err
	}

	err = blockchain.StoreBlockHashAtHeight(block.Hash(), block.Msg.Header.Height)
	if err != nil {
		return err
	}

	err = blockchain.StoreBlockHashAtHeight(block.Hash(), block.Msg.Header.Height)
	if err != nil {
		return err
	}

	return nil
}

func (blockchain *Blockchain) PopBlock(block *Block) error {
	longestChain, err := blockchain.FindLongestChain()
	if err != nil {
		return err
	}

	if longestChain.Hash() != block.Hash() {
		return errors.New("block must be the best block")
	}

	utxos, err := blockchain.FetchAllUtxos()
	if err != nil {
		return err
	}

	stxoj, err := blockchain.FetchStxojEntry(block.Hash())
	if err != nil {
		return err
	}

	var currentStxoId uint64

	for _, tx := range block.Txs {
		for index := range tx.Msg.Outputs {
			utxos.RemoveEntry(&network.Outpoint{
				Hash:  tx.Hash(),
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

	blockchain.StoreUtxos(utxos)
	blockchain.RemoveStxojEntry(block.Hash())

	blockchain.RemoveFollowing(block.Msg.Header.HashPrevBlock)
	blockchain.StoreLongestChain(block.Msg.Header.HashPrevBlock)

	return nil
}

func (blockchain *Blockchain) AcceptBlock(block *Block) (bool, error) {
	err := blockchain.StoreBlock(block)
	if err != nil {
		return false, err
	}

	longestChain, err := blockchain.FindLongestChain()
	if err != nil {
		return false, err
	}

	if longestChain.Hash().IsEqual(block.Msg.Header.HashPrevBlock) {
		err = blockchain.PushBlock(block)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return true, nil
}

func (blockchain *Blockchain) AddOrphan(block *Block) {
	blockchain.orphans[block.Hash()] = block
	blockchain.prevHashToOrphans[block.Msg.Header.HashPrevBlock] = append(blockchain.prevHashToOrphans[block.Msg.Header.HashPrevBlock], block.Hash())
}

func (blockchain *Blockchain) ProcessOrphans(acceptedBlock *Block) error {
	for _, blockHash := range blockchain.prevHashToOrphans[acceptedBlock.Hash()] {
		block, err := blockchain.FindOrphan(blockHash)
		if err != nil {
			return err
		}

		valid, err := blockchain.ValidateBlock(block)
		if err != nil {
			return err
		}
		if valid != nil {
			blockchain.RemoveOrphan(block.Hash())
		}

		_, err = blockchain.AcceptBlock(block)
		if err != nil {
			return err
		}
		err = blockchain.ProcessOrphans(block)
	}

	return nil
}

func (blockchain *Blockchain) FindOrphan(hash *utils.Hash) (*Block, error) {
	return blockchain.orphans[hash], nil
}

func (blockchain *Blockchain) RemoveOrphan(hash *utils.Hash) {
	for _, blockHash := range blockchain.prevHashToOrphans[hash] {
		blockchain.RemoveOrphan(blockHash)
	}

	delete(blockchain.prevHashToOrphans, hash)
	delete(blockchain.orphans, hash)
}

func (blockchain *Blockchain) ProcessBlock(block *Block) (error, error) {
	entry, err := blockchain.Index.FindBlock(block.Hash())
	if err != nil {
		return nil, err
	}
	if entry != nil {
		return errors.New("a block with the same hash exist"), nil
	}

	orphanBlock, err := blockchain.FindOrphan(block.Hash())
	if err != nil {
		return nil, nil
	}
	if orphanBlock != nil {
		return errors.New("an orphan block with the same hash exist"), nil
	}

	if valid := block.IsSane(); valid != nil {
		return errors.Wrap(valid, "the block is not sane"), nil
	}

	entry, err = blockchain.Index.FindBlock(block.Msg.Header.HashPrevBlock)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		log.Info("block is an orphan")

		blockchain.AddOrphan(block)

		return nil, nil
	}

	valid, err := blockchain.ValidateBlock(block)
	if err != nil {
		return nil, err
	}
	if valid != nil {
		return errors.Wrap(valid, "the block is invalid"), nil
	}

	_, err = blockchain.AcceptBlock(block)
	if err != nil {
		return nil, err
	}

	err = blockchain.ProcessOrphans(block)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
