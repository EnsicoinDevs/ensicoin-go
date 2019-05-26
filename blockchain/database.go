package blockchain

import (
	"bytes"
	"encoding/binary"
	"github.com/EnsicoinDevs/eccd/utils"
	bolt "github.com/etcd-io/bbolt"
)

func dbStoreBestBlock(tx *bolt.Tx, hash *utils.Hash) error {
	return tx.Bucket(statsBucket).Put([]byte("bestBlockHash"), hash.Bytes())
}

func dbStoreStxojEntry(tx *bolt.Tx, hash *utils.Hash, stxoj []*UtxoEntry) error {
	b := tx.Bucket(stxojBucket)

	buf := bytes.NewBuffer(nil)

	var err error

	for _, stxo := range stxoj {
		if err = WriteUtxoEntry(buf, stxo); err != nil {
			return err
		}
	}

	return b.Put(hash.Bytes(), buf.Bytes())
}

func dbRemoveStxojEntry(tx *bolt.Tx, hash *utils.Hash) error {
	return tx.Bucket(followingBucket).Delete(hash.Bytes())
}

func dbStoreUtxos(tx *bolt.Tx, utxos *Utxos) error {
	if err := tx.DeleteBucket(utxosBucket); err != nil {
		return err
	}

	b, err := tx.CreateBucket(utxosBucket)
	if err != nil {
		return err
	}

	for outpoint, entry := range utxos.entries {
		outpointBytes, err := outpoint.MarshalBinary()
		if err != nil {
			return err
		}

		entryBytes, err := entry.MarshalBinary()
		if err != nil {
			return err
		}

		if err = b.Put(outpointBytes, entryBytes); err != nil {
			return err
		}
	}

	return nil
}

func dbStoreBlockHashAtHeight(tx *bolt.Tx, hash *utils.Hash, height uint32) error {
	heightBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(heightBytes, height)

	return tx.Bucket(heightsBucket).Put(heightBytes, hash.Bytes())
}

func dbStoreFollowing(tx *bolt.Tx, prevHash *utils.Hash, hash *utils.Hash) error {
	return tx.Bucket(followingBucket).Put(prevHash.Bytes(), hash.Bytes())
}

func dbRemoveFollowing(tx *bolt.Tx, prevHash *utils.Hash) error {
	return tx.Bucket(followingBucket).Delete(prevHash.Bytes())
}
