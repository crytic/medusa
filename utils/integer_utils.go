package utils

import (
	"cmp"
	"math/big"

	"golang.org/x/exp/constraints"
)

// ConstrainIntegerToBounds takes a provided big integer and minimum/maximum bounds (inclusive) and ensures
// that the provided integer is represented in those bounds. In effect, this simulates overflow and underflow.
// The minimum must not exceed the maximum. Returns an independent copy of the constrained integer.
func ConstrainIntegerToBounds(b *big.Int, min *big.Int, max *big.Int) *big.Int {
	if b.Cmp(min) >= 0 && b.Cmp(max) <= 0 {
		return new(big.Int).Set(b)
	}

	width := new(big.Int).Sub(max, min)
	width.Add(width, big.NewInt(1))
	result := new(big.Int).Sub(b, min)
	// Euclidean modulo wraps negative inputs into [0, width) as well as positive ones.
	result.Mod(result, width)
	return result.Add(result, min)
}

// ConstrainIntegerToBitLength takes a provided big integer, signed indicator, and bit length and ensures that the
// provided integer is represented in those bounds. In effect, this simulates overflow and underflow.
// Returns the constrained integer.
func ConstrainIntegerToBitLength(b *big.Int, signed bool, bitLength int) *big.Int {
	// Calculate our min and max bounds for this integer.
	min, max := GetIntegerConstraints(signed, bitLength)

	// Constrain to the calculated bounds.
	return ConstrainIntegerToBounds(b, min, max)
}

// GetIntegerConstraints takes a given signed indicator and bit length for a prospective integer and determines the
// minimum/maximum value boundaries.
// Returns the minimum and maximum value for the provided integer properties. Minimums and maximums are inclusive.
func GetIntegerConstraints(signed bool, bitLength int) (*big.Int, *big.Int) {
	exponent := bitLength
	if signed {
		exponent--
	}
	// Match big.Int.Exp's result of one for a negative exponent without a modulus.
	if exponent < 0 {
		exponent = 0
	}
	max := new(big.Int).Lsh(big.NewInt(1), uint(exponent))
	min := new(big.Int)
	if signed {
		min.Neg(max)
	}
	max.Sub(max, big.NewInt(1))
	return min, max
}

// AbsDiff provides a way of taking the absolute difference between two integers
func AbsDiff[T constraints.Integer](x T, y T) T {
	if x >= y {
		return x - y
	} else {
		return y - x
	}
}

// Abs provides a way of taking the absolute value of an integer
func Abs[T constraints.Integer](x T) T {
	if x < 0 {
		return -x
	}
	return x
}

// Min provides generic support for various integer types to be compared and the minimum of two values returned.
// Returns the minimum of the two values provided.
func Min[T cmp.Ordered](x T, y T) T {
	if x <= y {
		return x
	}
	return y
}

// Max provides generic support for various integer types to be compared and the maximum of two values returned.
// Returns the maximum of the two values provided.
func Max[T cmp.Ordered](x T, y T) T {
	if x >= y {
		return x
	}
	return y
}
