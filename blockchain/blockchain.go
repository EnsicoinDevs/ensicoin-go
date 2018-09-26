package blockchain

import (
	"encoding/json"
	bolt "github.com/etcd-io/bbolt"
	"github.com/pkg/errors"
)

type Blockchain struct {
	db *bolt.DB
}

func NewBlockchain() *Blockchain {
	return &Blockchain{}
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
