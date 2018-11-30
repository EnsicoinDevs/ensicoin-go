package mempool

import (
	"github.com/EnsicoinDevs/ensicoincoin/blockchain"
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"github.com/EnsicoinDevs/ensicoincoin/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sync"
)

type Config struct {
	FetchUtxos func(tx *blockchain.Tx) (*blockchain.Utxos, []*network.Outpoint, error)
}

type Mempool struct {
	config Config

	txs              map[*utils.Hash]*blockchain.Tx
	orphans          map[*utils.Hash]*blockchain.Tx
	outpoints        map[*network.Outpoint]*blockchain.Tx
	orphansOutpoints map[*network.Outpoint]*blockchain.Tx
	mutex            sync.RWMutex
}

func NewMempool(config *Config) *Mempool {
	return &Mempool{
		config: *config,

		txs:              make(map[*utils.Hash]*blockchain.Tx),
		orphans:          make(map[*utils.Hash]*blockchain.Tx),
		outpoints:        make(map[*network.Outpoint]*blockchain.Tx),
		orphansOutpoints: make(map[*network.Outpoint]*blockchain.Tx),
	}
}

func (mempool *Mempool) addTx(tx *blockchain.Tx) {
	mempool.txs[tx.Hash()] = tx

	for _, input := range tx.Msg.Inputs {
		mempool.outpoints[input.PreviousOutput] = tx
	}

	log.WithField("tx", tx.Hash).Debug("accepted transaction")
}

func (mempool *Mempool) addOrphan(tx *blockchain.Tx) {
	mempool.orphans[tx.Hash()] = tx

	for _, input := range tx.Msg.Inputs {
		mempool.orphansOutpoints[input.PreviousOutput] = tx
	}

	log.WithField("tx", tx.Hash).Debug("accepted orphan transaction")
}

func (mempool *Mempool) findTxByHash(hash *utils.Hash) *blockchain.Tx {
	tx := mempool.txs[hash]
	if tx == nil {
		tx = mempool.orphans[hash]
	}

	return tx
}

func (mempool *Mempool) fetchUtxos(tx *blockchain.Tx) (*blockchain.Utxos, []*network.Outpoint, error) {
	utxos, missings, err := mempool.config.FetchUtxos(tx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error fetching utxos")
	}

	var missingsAfter []*network.Outpoint

	for _, missingOutpoint := range missings {
		spentTx := mempool.outpoints[missingOutpoint]
		if spentTx == nil {
			missingsAfter = append(missingsAfter, missingOutpoint)
		} else {
			utxos.AddEntryWithTx(missingOutpoint, spentTx, 0)
		}
	}

	return utxos, missingsAfter, nil
}

func (mempool *Mempool) validateTx(tx *blockchain.Tx) (bool, []*utils.Hash) {
	if !tx.IsSane() {
		return false, nil
	}

	if tx.IsCoinBase() {
		return false, nil
	}

	if mempool.findTxByHash(tx.Hash()) != nil {
		return false, nil
	}

	utxos, missings, _ := mempool.fetchUtxos(tx)
	if len(missings) != 0 {
		encountered := make(map[*utils.Hash]bool)
		for _, outpoint := range missings {
			encountered[outpoint.Hash] = true
		}

		var missingParents []*utils.Hash
		for txHash, _ := range encountered {
			missingParents = append(missingParents, txHash)
		}

		return true, missingParents
	}

	_ = utxos

	return true, nil
}

func (mempool *Mempool) processOrphans(tx *blockchain.Tx) []*utils.Hash {
	outpoint := &network.Outpoint{
		Hash: tx.Hash(),
	}

	var acceptedTxs []*utils.Hash

	for outputId := range tx.Msg.Outputs {
		outpoint.Index = uint32(outputId)

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

			acceptedTxs = append(acceptedTxs, orphan.Hash())
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

func (mempool *Mempool) processTx(tx *blockchain.Tx) []*utils.Hash {
	valid, missingParents := mempool.validateTx(tx)

	if !valid {
		return nil
	}

	if len(missingParents) == 0 {
		var acceptedTxs []*utils.Hash

		acceptedTxs = append(acceptedTxs, tx.Hash())
		mempool.addTx(tx)
		acceptedTxs = append(acceptedTxs, mempool.processOrphans(tx)...)

		return acceptedTxs
	}

	mempool.addOrphan(tx)
	return nil
}

func (mempool *Mempool) ProcessTx(tx *blockchain.Tx) []*utils.Hash {
	mempool.mutex.Lock()
	defer mempool.mutex.Unlock()

	return mempool.processTx(tx)
}

func (mempool *Mempool) FindTxByHash(hash *utils.Hash) *blockchain.Tx {
	mempool.mutex.RLock()
	defer mempool.mutex.RUnlock()

	return mempool.findTxByHash(hash)
}

func (mempool *Mempool) removeOrphan(tx *blockchain.Tx, recursively bool) {
	_, exists := mempool.orphans[tx.Hash()]
	if !exists {
		return
	}

	for _, input := range tx.Msg.Inputs {
		delete(mempool.orphansOutpoints, input.PreviousOutput)
	}

	if recursively {
		outpoint := &network.Outpoint{
			Hash: tx.Hash(),
		}

		for outputId := range tx.Msg.Outputs {
			outpoint.Index = uint32(outputId)
			orphan, exists := mempool.orphansOutpoints[outpoint]
			if exists {
				mempool.removeOrphan(orphan, true)
			}
		}
	}

	delete(mempool.orphans, tx.Hash())
}
