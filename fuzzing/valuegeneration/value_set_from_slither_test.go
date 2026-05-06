package valuegeneration

import (
	"math/big"
	"testing"

	"github.com/crytic/medusa-geth/common"
	compilationTypes "github.com/crytic/medusa/compilation/types"
	"github.com/stretchr/testify/assert"
)

// TestSeedFromSlitherInvalidUintDoesNotPanic asserts that a numeric-typed Slither
// constant whose value is not a valid base-10 string is skipped instead of being
// fed into big.Int operations as a nil pointer.
//
// On stock master this test panics in (*big.Int).Neg via SeedFromSlither.
func TestSeedFromSlitherInvalidUintDoesNotPanic(t *testing.T) {
	vs := NewValueSet()

	assert.NotPanics(t, func() {
		vs.SeedFromSlither(&compilationTypes.SlitherResults{
			Constants: []compilationTypes.Constant{
				{Type: "uint256", Value: "b'\\x01'"},
			},
		})
	})

	assert.Empty(t, vs.Integers(), "no integers should be seeded from an unparsable constant")
	assert.Empty(t, vs.Bytes(), "no bytes should be seeded from an unparsable constant")
}

// TestSeedFromSlitherValidUintSeeds asserts that a well-formed uint256 constant
// continues to seed the value set with the value, its negation, and its bytes,
// preserving the existing decimal-only parsing behavior.
func TestSeedFromSlitherValidUintSeeds(t *testing.T) {
	vs := NewValueSet()

	vs.SeedFromSlither(&compilationTypes.SlitherResults{
		Constants: []compilationTypes.Constant{
			{Type: "uint256", Value: "42"},
		},
	})

	assert.True(t, vs.ContainsInteger(big.NewInt(42)), "value set must contain 42")
	assert.True(t, vs.ContainsInteger(big.NewInt(-42)), "value set must contain -42")
	assert.True(t, vs.ContainsBytes(big.NewInt(42).Bytes()), "value set must contain []byte{42}")
}

// TestSeedFromSlitherInvalidAddressDoesNotPanic asserts that an address-typed
// Slither constant whose value is not a valid base-10 string is skipped instead
// of being passed to common.BigToAddress as a nil pointer.
//
// On stock master this test panics inside common.BigToAddress -> b.Bytes().
func TestSeedFromSlitherInvalidAddressDoesNotPanic(t *testing.T) {
	vs := NewValueSet()

	assert.NotPanics(t, func() {
		vs.SeedFromSlither(&compilationTypes.SlitherResults{
			Constants: []compilationTypes.Constant{
				{Type: "address", Value: "not-a-decimal-address"},
			},
		})
	})

	assert.Empty(t, vs.Addresses(), "no address should be seeded from an unparsable constant")
	assert.Empty(t, vs.Bytes(), "no bytes should be seeded from an unparsable constant")
}

// TestSeedFromSlitherValidAddressSeeds asserts that a well-formed address
// constant (decimal big-int representation) still seeds the corresponding
// common.Address into the value set.
func TestSeedFromSlitherValidAddressSeeds(t *testing.T) {
	vs := NewValueSet()

	vs.SeedFromSlither(&compilationTypes.SlitherResults{
		Constants: []compilationTypes.Constant{
			{Type: "address", Value: "42"},
		},
	})

	assert.True(t, vs.ContainsAddress(common.BigToAddress(big.NewInt(42))),
		"value set must contain BigToAddress(42)")
}
