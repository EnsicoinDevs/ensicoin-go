package blockchain

import (
	"crypto/sha256"
	"github.com/EnsicoinDevs/eccd/utils"
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
		if len(hashes)%2 != 0 {
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
	hash := utils.Hash(sha256.Sum256(append(a.Bytes(), b.Bytes()...)))

	hash = sha256.Sum256(hash.Bytes())

	return &hash
}
