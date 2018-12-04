package utils

import (
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
