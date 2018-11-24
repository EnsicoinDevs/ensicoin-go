package utils

type Hash [32]byte

func NewHash(src []byte) *Hash {
	var hash Hash

	copy(hash[:], src)

	return &hash
}
