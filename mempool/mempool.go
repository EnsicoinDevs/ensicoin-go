package mempool

import (
	"github.com/EnsicoinDevs/ensicoin-go/blockchain"
	"sync"
)

type Mempool struct {
	txs   map[string]*blockchain.Tx
	mutex sync.RWMutex
}

func NewMempool() *Mempool {
	return &Mempool{
		txs: make(map[string]*blockchain.Tx),
	}
}

func (mempool *Mempool) addTx(tx *blockchain.Tx) {
	mempool.mutex.Lock()
	mempool.txs[tx.Hash] = tx
	mempool.mutex.Unlock()
}

func (mempool *Mempool) ProcessTx(tx *blockchain.Tx) bool {
	if !tx.IsSane() {
		return false
	}

	if tx.IsCoinBase() {
		return false
	}

	mempool.addTx(tx)

	return true
}

func (mempool *Mempool) FindTxByHash(hash string) *blockchain.Tx {
	mempool.mutex.RLock()
	defer mempool.mutex.RUnlock()

	return mempool.txs[hash]
}

func (mempool *Mempool) RemoveTxByHash(hash string) {
	mempool.mutex.Lock()
	delete(mempool.txs, hash)
	mempool.mutex.Unlock()
}
