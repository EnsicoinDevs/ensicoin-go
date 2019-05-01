package utils

import (
	"math/big"
)

func BitsToBig(bits uint32) *big.Int {
	mantissa := int64(bits & 0x00ffffff)
	exponent := uint(bits >> 24)

	if exponent <= 3 {
		mantissa >>= 8 * (3 - exponent)
		return big.NewInt(mantissa)
	}

	target := big.NewInt(mantissa)
	return target.Lsh(target, 8*(exponent-3))
}

func BigToBits(target *big.Int) uint32 {
	var mantissa uint32
	exponent := uint(len(target.Bytes()))

	if exponent <= 3 {
		mantissa = uint32(target.Bits()[0])
		mantissa <<= 8 * (3 - exponent)
	} else {
		targetCopy := new(big.Int).Set(target)
		mantissa = uint32(targetCopy.Rsh(targetCopy, 8*(exponent-3)).Bits()[0])
	}

	return uint32(exponent<<24) | mantissa
}

func HashToBig(hash *Hash) *big.Int {
	return new(big.Int).SetBytes(hash[:])
}
