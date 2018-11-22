package mempool

import (
	"github.com/EnsicoinDevs/ensicoincoin/blockchain"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sync"
)

type Config struct {
	FetchUtxos func(tx *blockchain.Tx) (*blockchain.Utxos, []*blockchain.TxOutpoint, error)
}

type Mempool struct {
	config Config

	txs              map[string]*blockchain.Tx
	orphans          map[string]*blockchain.Tx
	outpoints        map[blockchain.TxOutpoint]*blockchain.Tx
	orphansOutpoints map[blockchain.TxOutpoint]*blockchain.Tx
	mutex            sync.RWMutex
}

func NewMempool(config *Config) *Mempool {
	return &Mempool{
		config: *config,

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

func (mempool *Mempool) fetchUtxos(tx *blockchain.Tx) (*blockchain.Utxos, []*blockchain.TxOutpoint, error) {
	utxos, missings, err := mempool.config.FetchUtxos(tx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error fetching utxos")
	}

	var missingsAfter []*blockchain.TxOutpoint

	for _, missingOutpoint := range missings {
		spentTx := mempool.outpoints[*missingOutpoint]
		if spentTx == nil {
			missingsAfter = append(missingsAfter, missingOutpoint)
		} else {
			utxos.AddEntryWithTx(*missingOutpoint, spentTx, -1)
		}
	}

	return utxos, missingsAfter, nil
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

	utxos, missings, _ := mempool.fetchUtxos(tx)
	if len(missings) != 0 {
		encountered := make(map[string]bool)
		for _, outpoint := range missings {
			encountered[outpoint.TxHash] = true
		}

		var missingParents []string
		for txHash, _ := range encountered {
			missingParents = append(missingParents, txHash)
		}

		return true, missingParents
	}

	_ = utxos

	return true, nil
}

func (mempool *Mempool) processOrphans(tx *blockchain.Tx) []string {
	outpoint := blockchain.TxOutpoint{
		TxHash: tx.Hash,
	}

	var acceptedTxs []string

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

			acceptedTxs = append(acceptedTxs, orphan.Hash)
			mempool.addTx(orphan)
			mempool.removeOrphan(orphan, false)
			acceptedTxs = append(acceptedTxs, mempool.processOrphans(orphan)...)
		}
	}

	return acceptedTxs
}

func (mempool *Mempool) ProcessOrphans(tx *blockchain.Tx) {
	mempool.mutex.Lock()

	mempool.processOrphans(tx)

	mempool.mutex.Unlock()
}

func (mempool *Mempool) processTx(tx *blockchain.Tx) []string {
	valid, missingParents := mempool.validateTx(tx)

	if !valid {
		return nil
	}

	if len(missingParents) == 0 {
		var acceptedTxs []string

		acceptedTxs = append(acceptedTxs, tx.Hash)
		mempool.addTx(tx)
		acceptedTxs = append(acceptedTxs, mempool.processOrphans(tx)...)

		return acceptedTxs
	}

	mempool.addOrphan(tx)
	return nil
}

func (mempool *Mempool) ProcessTx(tx *blockchain.Tx) []string {
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
