package mempool

import (
	"github.com/johynpapin/ensicoin-go/blockchain"
	"sync"
)

type Mempool struct {
	transactions map[string]*blockchain.Transaction
	mutex        sync.RWMutex
}

func NewMempool() *Mempool {
	return &Mempool{
		transactions: make(map[string]*blockchain.Transaction),
	}
}

func (mempool *Mempool) AddTransaction(tx *blockchain.Transaction) {
	mempool.mutex.Lock()
	mempool.transactions[tx.Hash] = tx
	mempool.mutex.Unlock()
}

func (mempool *Mempool) FindTransactionByHash(hash string) *blockchain.Transaction {
	mempool.mutex.RLock()
	defer mempool.mutex.RUnlock()

	return mempool.transactions[hash]
}

func (mempool *Mempool) RemoveTransactionByHash(hash string) {
	mempool.mutex.Lock()
	delete(mempool.transactions, hash)
	mempool.mutex.Unlock()
}
