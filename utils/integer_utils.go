package utils

import (
	"golang.org/x/exp/constraints"
	"math/big"
)

// ConstrainIntegerToBounds takes a provided big integer and minimum/maximum bounds (inclusive) and ensures
// that the provided integer is represented in those bounds. In effect, this simulates overflow and underflow.
// Returns the constrained integer.
func ConstrainIntegerToBounds(b *big.Int, min *big.Int, max *big.Int) *big.Int {
	// Get the bounding range
	boundingRange := big.NewInt(0).Add(big.NewInt(0).Sub(max, min), big.NewInt(1))

	// Next we check boundaries for underflow/overflow. If it occurred, we calculate the distance and then find out
	// how many wrap-arounds (bounding ranges) should be added/subtracted to correct the value. This is done by
	// division with ceiling: (distance + (boundingRange - 1)) / distance. This way even a small underflow like -1 in
	// an unsigned int (meaning underflow by 1) will result in one bounding range being added to wrap back around.

	// Check underflow
	if b.Cmp(min) < 0 {
		distance := big.NewInt(0).Sub(min, b)
		correction := big.NewInt(0).Div(big.NewInt(0).Add(distance, big.NewInt(0).Sub(boundingRange, big.NewInt(1))), boundingRange)
		correction.Mul(correction, boundingRange)
		return big.NewInt(0).Add(b, correction)
	}

	// Check overflow
	if b.Cmp(max) > 0 {
		distance := big.NewInt(0).Sub(b, max)
		correction := big.NewInt(0).Div(big.NewInt(0).Add(distance, big.NewInt(0).Sub(boundingRange, big.NewInt(1))), boundingRange)
		correction.Mul(correction, boundingRange)
		return big.NewInt(0).Sub(b, correction)
	}

	// b is in range, return a copy of it
	return big.NewInt(0).Set(b)
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
	// Calculate our min and max bounds for this integer.
	var min, max *big.Int
	if signed {
		// Set max as 2^(bitLen - 1) - 1
		max = big.NewInt(2)
		max.Exp(max, big.NewInt(int64(bitLength-1)), nil)
		max.Sub(max, big.NewInt(1))

		// Set min as -(2^(bitLen - 1))
		min = big.NewInt(0).Mul(max, big.NewInt(-1))
		min.Sub(min, big.NewInt(1))
	} else {
		// Set minimum as 2^bitLen - 1
		max = big.NewInt(2)
		max.Exp(max, big.NewInt(int64(bitLength)), nil)
		max.Sub(max, big.NewInt(1))

		// Set minimum as zero
		min = big.NewInt(0)
	}
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
