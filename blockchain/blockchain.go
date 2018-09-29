package blockchain

import (
	"encoding/json"
	bolt "github.com/etcd-io/bbolt"
	"github.com/pkg/errors"
)

type Blockchain struct {
	db *bolt.DB

	GenesisBlock *Block
}

func NewBlockchain() *Blockchain {
	return &Blockchain{
		GenesisBlock: &genesisBlock,
	}
}

func (blockchain *Blockchain) Load() error {
	var err error
	blockchain.db, err = bolt.Open("blockchain.db", 0600, nil)
	if err != nil {
		return errors.Wrap(err, "error opening the blockchain database")
	}

	genesisBlock.ComputeHash()

	shouldStoreGenesisBlock := false

	blockchain.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("blocks"))
		if err != nil {
			return errors.Wrap(err, "error creating the blocks bucket")
		}

		genesisBlockInDb := b.Get([]byte(genesisBlock.Hash))
		if genesisBlockInDb == nil {
			shouldStoreGenesisBlock = true
		}

		b, err = tx.CreateBucketIfNotExists([]byte("stats"))
		if err != nil {
			return errors.Wrap(err, "error creating the stats bucket")
		}

		longestChainInDb := b.Get([]byte("longestChain"))
		if longestChainInDb == nil {
			b.Put([]byte("longestChain"), []byte(genesisBlock.Hash))
		}

		_, err = tx.CreateBucketIfNotExists([]byte("following"))
		if err != nil {
			return errors.Wrap(err, "error creating the following bucket")
		}

		return nil
	})

	if shouldStoreGenesisBlock {
		blockchain.StoreBlock(&genesisBlock)
	}

	return nil
}

func (blockchain *Blockchain) StoreBlock(block *Block) {
	if block.Hash == "" {
		block.ComputeHash()
	}

	blockchain.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("blocks"))

		blockBytes, err := json.Marshal(block)
		if err != nil {
			return errors.Wrap(err, "error marshaling the block")
		}

		b.Put([]byte(block.Hash), blockBytes)

		return nil
	})
}

func (blockchain *Blockchain) FindBlockByHash(hash string) (*Block, error) {
	var block *Block

	err := blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("blocks"))

		blockBytes := b.Get([]byte(hash))
		if blockBytes == nil {
			return nil
		}

		if err := json.Unmarshal(blockBytes, &block); err != nil {
			return errors.Wrap(err, "error unmarshaling a block")
		}

		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "error finding a block")
	}

	return block, nil
}

func (blockchain *Blockchain) FindLongestChain() (*Block, error) {
	var longestChainHash string

	err := blockchain.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("stats"))
		longestChainHash = string(b.Get([]byte("longestChain")))

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

func (blockchain *Blockchain) FindBlockHashesStartingAt(hash string) ([]string, error) {
	var hashes []string

	err := blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("following"))

		c := b.Cursor()

		current := []byte(hash)

		for k, v := c.Seek([]byte(hash)); k != nil; k, v = c.Seek(current) {
			hashes = append(hashes, string(v))

			current = v
		}

		return nil
	})
	if err != nil {
		return nil, errors.Wrap(nil, "error finding all the hashes")
	}

	return hashes, nil
}
