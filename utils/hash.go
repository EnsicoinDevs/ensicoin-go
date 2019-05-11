package utils

import (
	"encoding/hex"
	"reflect"
)

type Hash [32]byte

func NewHash(src []byte) *Hash {
	var hash Hash

	copy(hash[:], src)

	return &hash
}

func (hash *Hash) IsEqual(otherHash *Hash) bool {
	return reflect.DeepEqual(hash, otherHash)
}

func (hash *Hash) String() string {
	return hex.EncodeToString(hash[:])
}
