package utils

import (
	"encoding/hex"
	"math/big"
	"reflect"
)

type Hash [32]byte

func NewHash(src []byte) *Hash {
	var hash Hash

	copy(hash[:], src)

	return &hash
}

func (hash *Hash) Big() *big.Int {
	return new(big.Int).SetBytes(hash[:])
}

func (hash *Hash) IsEqual(otherHash *Hash) bool {
	return reflect.DeepEqual(hash, otherHash)
}

func (hash *Hash) Bytes() []byte {
	return hash[:]
}

func (hash *Hash) String() string {
	return hex.EncodeToString(hash[:])
}

func BigToHash(big *big.Int) *Hash {
	return NewHash(big.Bytes())
}
