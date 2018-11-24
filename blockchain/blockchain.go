package blockchain

import (
	"encoding/json"
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"github.com/EnsicoinDevs/ensicoincoin/utils"
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

	shouldStoreGenesisBlock := false

	blockchain.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("blocks"))
		if err != nil {
			return errors.Wrap(err, "error creating the blocks bucket")
		}

		genesisBlockInDb := b.Get(genesisBlock.Hash()[:])
		if genesisBlockInDb == nil {
			shouldStoreGenesisBlock = true
		}

		b, err = tx.CreateBucketIfNotExists([]byte("stats"))
		if err != nil {
			return errors.Wrap(err, "error creating the stats bucket")
		}

		longestChainInDb := b.Get([]byte("longestChain"))
		if longestChainInDb == nil {
			b.Put([]byte("longestChain"), genesisBlock.Hash()[:])
		}

		_, err = tx.CreateBucketIfNotExists([]byte("following"))
		if err != nil {
			return errors.Wrap(err, "error creating the following bucket")
		}

		_, err = tx.CreateBucketIfNotExists([]byte("utxos"))
		if err != nil {
			return errors.Wrap(err, "error creating the utxos bucket")
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
		b := btx.Bucket([]byte("utxos"))

		for _, input := range tx.Msg.Inputs {
			outpoint := input.PreviousOutput
			outpointBytes, err := json.Marshal(outpoint)
			if err != nil {
				return errors.Wrap(err, "error marshaling an outpoint")
			}

			utxoBytes := b.Get(outpointBytes)
			if utxoBytes == nil {
				missings = append(missings, outpoint)
				continue
			}

			var utxo *UtxoEntry
			if err = json.Unmarshal(utxoBytes, &utxo); err != nil {
				return errors.Wrap(err, "error unmarshaling an utxo entry")
			}

			utxos.AddEntry(outpoint, utxo)
		}

		return nil
	})

	return utxos, missings, err
}

func (blockchain *Blockchain) StoreBlock(block *Block) error {
	err := blockchain.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("blocks"))

		blockBytes, err := json.Marshal(block)
		if err != nil {
			return errors.Wrap(err, "error marshaling the block")
		}

		b.Put(block.Hash()[:], blockBytes)

		return nil
	})

	return err
}

func (blockchain *Blockchain) FindBlockByHash(hash *utils.Hash) (*Block, error) {
	var block *Block

	err := blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("blocks"))

		blockBytes := b.Get(hash[:])
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
	var longestChainHash *utils.Hash

	err := blockchain.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("stats"))
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
		b := tx.Bucket([]byte("following"))

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

func (blockchain *Blockchain) HandleBlockMessage() {

}
