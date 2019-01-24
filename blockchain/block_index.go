package blockchain

import (
	"github.com/EnsicoinDevs/ensicoincoin/utils"
	"sync"
	"time"
)

const MAX_ENTRY_IN_BLOCK_INDEX = 4032

type BlockIndexEntry struct {
	hash          *utils.Hash
	hashPrevBlock *utils.Hash
	height        uint32
	timestamp     time.Time
	bits          uint32
}

type BlockIndex struct {
	blockchain *Blockchain

	mutex     sync.RWMutex
	index     map[*utils.Hash]*BlockIndexEntry
	heights   map[uint32][]*utils.Hash
	minHeight uint32
}

func NewBlockIndex(blockchain *Blockchain) *BlockIndex {
	return &BlockIndex{
		blockchain: blockchain,

		index:   make(map[*utils.Hash]*BlockIndexEntry),
		heights: make(map[uint32][]*utils.Hash),
	}
}

func (index *BlockIndex) findBlock(hash *utils.Hash) (*BlockIndexEntry, error) {
	index.mutex.RLock()
	entry, exist := index.index[hash]
	index.mutex.RUnlock()
	if exist {
		return entry, nil
	}

	block, err := index.blockchain.findBlockByHash(hash)
	if err != nil {
		return nil, err
	}

	if block == nil {
		return nil, nil
	}

	entry = &BlockIndexEntry{
		hash:          block.Hash(),
		hashPrevBlock: block.Msg.Header.HashPrevBlock,
		height:        block.Msg.Header.Height,
		timestamp:     block.Msg.Header.Timestamp,
		bits:          block.Msg.Header.Bits,
	}

	index.addEntry(entry)

	return entry, nil
}

func (index *BlockIndex) FindBlock(hash *utils.Hash) (*BlockIndexEntry, error) {
	index.blockchain.lock.RLock()
	defer index.blockchain.lock.RUnlock()

	return index.findBlock(hash)
}

func (index *BlockIndex) addEntry(entry *BlockIndexEntry) {
	index.mutex.Lock()

	if len(index.index) == 0 || entry.height < index.minHeight {
		index.minHeight = entry.height
	}
	index.index[entry.hash] = entry
	index.heights[entry.height] = append(index.heights[entry.height], entry.hash)

	index.ensureMaxSize()

	index.mutex.Unlock()
}

func (index *BlockIndex) ensureMaxSize() {
	for len(index.index) > MAX_ENTRY_IN_BLOCK_INDEX {
		for _, hash := range index.heights[index.minHeight] {
			delete(index.index, hash)
		}

		delete(index.heights, index.minHeight)
		index.minHeight++
	}
}

func (index *BlockIndex) findAncestor(block *Block, height uint32) (*BlockIndexEntry, error) {
	entry := new(BlockIndexEntry)
	var err error

	entry.height = block.Msg.Header.Height
	entry.hashPrevBlock = block.Msg.Header.HashPrevBlock

	for entry.height != height {
		entry, err = index.findBlock(entry.hashPrevBlock)
		if err != nil {
			return nil, err
		}
	}

	return entry, nil
}
