package blockchain

import (
	"github.com/EnsicoinDevs/ensicoincoin/utils"
	"sync"
	"time"
)

const MAX_ENTRY_IN_BLOCK_INDEX = 4032

type blockIndexEntry struct {
	hash          *utils.Hash
	hashPrevBlock *utils.Hash
	height        uint32
	timestamp     time.Time
	bits          uint32
}

type blockIndex struct {
	blockchain *Blockchain

	mutex     sync.RWMutex
	index     map[*utils.Hash]*blockIndexEntry
	heights   map[uint32][]*utils.Hash
	minHeight uint32
}

func newBlockIndex(blockchain *Blockchain) *blockIndex {
	return &blockIndex{
		blockchain: blockchain,

		index:   make(map[*utils.Hash]*blockIndexEntry),
		heights: make(map[uint32][]*utils.Hash),
	}
}

func (index *blockIndex) findBlock(hash *utils.Hash) (*blockIndexEntry, error) {
	index.mutex.RLock()
	entry, exist := index.index[hash]
	index.mutex.RUnlock()
	if exist {
		return entry, nil
	}

	block, err := index.blockchain.FindBlockByHash(hash)
	if err != nil {
		return nil, err
	}

	if block == nil {
		return nil, nil
	}

	entry = &blockIndexEntry{
		hash:          block.Hash(),
		hashPrevBlock: block.Msg.Header.HashPrevBlock,
		height:        block.Msg.Header.Height,
		timestamp:     block.Msg.Header.Timestamp,
		bits:          block.Msg.Header.Bits,
	}

	index.addEntry(entry)

	return entry, nil
}

func (index *blockIndex) addEntry(entry *blockIndexEntry) {
	index.mutex.Lock()

	if len(index.index) == 0 || entry.height < index.minHeight {
		index.minHeight = entry.height
	}
	index.index[entry.hash] = entry
	index.heights[entry.height] = append(index.heights[entry.height], entry.hash)

	index.ensureMaxSize()

	index.mutex.Unlock()
}

func (index *blockIndex) ensureMaxSize() {
	for len(index.index) > MAX_ENTRY_IN_BLOCK_INDEX {
		for _, hash := range index.heights[index.minHeight] {
			delete(index.index, hash)
		}

		delete(index.heights, index.minHeight)
		index.minHeight++
	}
}

func (index *blockIndex) findAncestor(block *Block, height uint32) (*blockIndexEntry, error) {
	entry, err := index.findBlock(block.Hash())
	if err != nil {
		return nil, err
	}

	for entry.height != height {
		entry, err = index.findBlock(entry.hashPrevBlock)
		if err != nil {
			return nil, err
		}
	}

	return entry, nil
}
