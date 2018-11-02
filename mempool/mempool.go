package mempool

import (
	"github.com/EnsicoinDevs/ensicoincoin/blockchain"
	log "github.com/sirupsen/logrus"
	"sync"
)

type Mempool struct {
	txs              map[string]*blockchain.Tx
	orphans          map[string]*blockchain.Tx
	outpoints        map[blockchain.TxOutpoint]*blockchain.Tx
	orphansOutpoints map[blockchain.TxOutpoint]*blockchain.Tx
	mutex            sync.RWMutex
}

func NewMempool() *Mempool {
	return &Mempool{
		txs:              make(map[string]*blockchain.Tx),
		orphans:          make(map[string]*blockchain.Tx),
		outpoints:        make(map[blockchain.TxOutpoint]*blockchain.Tx),
		orphansOutpoints: make(map[blockchain.TxOutpoint]*blockchain.Tx),
	}
}

func (mempool *Mempool) addTx(tx *blockchain.Tx) {
	mempool.txs[tx.Hash] = tx

	for _, input := range tx.Inputs {
		mempool.outpoints[input.PreviousOutput] = tx
	}

	log.WithField("tx", tx.Hash).Debug("accepted transaction")
}

func (mempool *Mempool) addOrphan(tx *blockchain.Tx) {
	mempool.orphans[tx.Hash] = tx

	for _, input := range tx.Inputs {
		mempool.orphansOutpoints[input.PreviousOutput] = tx
	}

	log.WithField("tx", tx.Hash).Debug("accepted orphan transaction")
}

func (mempool *Mempool) findTxByHash(hash string) *blockchain.Tx {
	tx := mempool.txs[hash]
	if tx == nil {
		tx = mempool.orphans[hash]
	}

	return tx
}

func (mempool *Mempool) validateTx(tx *blockchain.Tx) (bool, []string) {
	if !tx.IsSane() {
		return false, nil
	}

	if tx.IsCoinBase() {
		return false, nil
	}

	if mempool.findTxByHash(tx.Hash) != nil {
		return false, nil
	}

	return true, nil
}

func (mempool *Mempool) processOrphans(tx *blockchain.Tx) {
	outpoint := blockchain.TxOutpoint{
		TxHash: tx.Hash,
	}

	for outputId := range tx.Outputs {
		outpoint.Index = uint(outputId)

		orphan, exists := mempool.orphansOutpoints[outpoint]
		if exists {
			valid, missingParents := mempool.validateTx(orphan)
			if !valid {
				mempool.removeOrphan(orphan, true)
				continue
			}

			if len(missingParents) != 0 {
				continue
			}

			mempool.addTx(orphan)
			mempool.removeOrphan(orphan, false)
			mempool.processOrphans(orphan)
		}
	}
}

func (mempool *Mempool) ProcessOrphans(tx *blockchain.Tx) {
	mempool.mutex.Lock()

	mempool.processOrphans(tx)

	mempool.mutex.Unlock()
}

func (mempool *Mempool) processTx(tx *blockchain.Tx) bool {
	valid, missingParents := mempool.validateTx(tx)

	if !valid {
		return false
	}

	mempool.addTx(tx)

	if len(missingParents) == 0 {
		mempool.processOrphans(tx)
	}

	mempool.addOrphan(tx)

	return true
}

func (mempool *Mempool) ProcessTx(tx *blockchain.Tx) bool {
	mempool.mutex.Lock()
	defer mempool.mutex.Unlock()

	return mempool.processTx(tx)
}

func (mempool *Mempool) FindTxByHash(hash string) *blockchain.Tx {
	mempool.mutex.RLock()
	defer mempool.mutex.RUnlock()

	return mempool.findTxByHash(hash)
}

func (mempool *Mempool) removeOrphan(tx *blockchain.Tx, recursively bool) {
	_, exists := mempool.orphans[tx.Hash]
	if !exists {
		return
	}

	for _, input := range tx.Inputs {
		delete(mempool.orphansOutpoints, input.PreviousOutput)
	}

	if recursively {
		outpoint := blockchain.TxOutpoint{
			TxHash: tx.Hash,
		}

		for outputId := range tx.Outputs {
			outpoint.Index = uint(outputId)
			orphan, exists := mempool.orphansOutpoints[outpoint]
			if exists {
				mempool.removeOrphan(orphan, true)
			}
		}
	}

	delete(mempool.orphans, tx.Hash)
}
