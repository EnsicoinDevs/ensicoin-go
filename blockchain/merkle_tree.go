package blockchain

import (
	"crypto/sha256"
	"github.com/EnsicoinDevs/ensicoincoin/utils"
	log "github.com/sirupsen/logrus"
)

func ComputeMerkleRoot(hashes []*utils.Hash) *utils.Hash {
	if len(hashes) == 0 {
		hash := sha256.Sum256(nil)
		return utils.NewHash(hash[:])
	}

	if len(hashes) == 1 {
		hashes = append(hashes, hashes[0])
	}

	for len(hashes) > 1 {
		log.WithField("len(hashes)", len(hashes)).Debug("next iteration")

		if len(hashes)%2 != 0 {
			log.Debug("\tduplicating the last hash")
			hashes = append(hashes, hashes[len(hashes)-1])
		}

		var leftHash *utils.Hash
		for i, hash := range hashes {
			if i%2 != 0 {
				hashes[((i+1)/2)-1] = DoubleHash(leftHash, hash)
			} else {
				leftHash = hash
			}
		}

		hashes = hashes[:len(hashes)/2]
	}

	return hashes[0]
}

func DoubleHash(a, b *utils.Hash) *utils.Hash {
	hash := utils.Hash(sha256.Sum256(append(a[:], b[:]...)))

	return &hash
}
