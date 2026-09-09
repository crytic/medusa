package utils

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConstrainIntegerToBounds(t *testing.T) {
	t.Parallel()
	for min := int64(-20); min <= 20; min++ {
		for width := int64(1); width <= 20; width++ {
			for input := int64(-100); input <= 100; input++ {
				lo, hi, value := big.NewInt(min), big.NewInt(min+width-1), big.NewInt(input)
				expected := (input - min) % width
				if expected < 0 {
					expected += width
				}
				actual := ConstrainIntegerToBounds(value, lo, hi)
				require.Equal(t, expected+min, actual.Int64())
				actual.SetInt64(999)
				require.Equal(t, input, value.Int64())
				require.Equal(t, min, lo.Int64())
				require.Equal(t, min+width-1, hi.Int64())
			}
		}
	}
}

func FuzzConstrainIntegerToBounds(f *testing.F) {
	f.Add([]byte{0xff, 0xff}, int64(-128), uint64(256), true)
	f.Add([]byte{0}, int64(42), uint64(1), false)
	f.Fuzz(func(t *testing.T, magnitude []byte, lower int64, span uint64, negative bool) {
		value := new(big.Int).SetBytes(magnitude)
		if negative {
			value.Neg(value)
		}
		min := big.NewInt(lower)
		width := new(big.Int).SetUint64(span)
		width.Add(width, big.NewInt(1))
		max := new(big.Int).Add(min, width)
		max.Sub(max, big.NewInt(1))
		result := ConstrainIntegerToBounds(value, min, max)
		require.GreaterOrEqual(t, result.Cmp(min), 0)
		require.LessOrEqual(t, result.Cmp(max), 0)
		difference := new(big.Int).Sub(value, result)
		require.Zero(t, difference.Rem(difference, width).Sign())
		require.Zero(t, ConstrainIntegerToBounds(result, min, max).Cmp(result))
	})
}

func TestIntegerConstraints(t *testing.T) {
	t.Parallel()
	for width := -2; width <= 1024; width++ {
		for _, signed := range []bool{false, true} {
			exponent := width
			if signed {
				exponent--
			}
			power := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(exponent)), nil)
			min, max := GetIntegerConstraints(signed, width)
			require.Zero(t, new(big.Int).Sub(power, big.NewInt(1)).Cmp(max))
			if signed {
				require.Zero(t, new(big.Int).Neg(power).Cmp(min))
			} else {
				require.Zero(t, min.Sign())
			}
		}
	}
}

// BenchmarkConstrainInteger measures bounded values and multi-width overflow and underflow.
func BenchmarkConstrainInteger(b *testing.B) {
	min, max := GetIntegerConstraints(true, 256)
	for _, name := range []string{"inRange", "overflow", "underflow"} {
		b.Run(name, func(b *testing.B) {
			value := big.NewInt(42)
			if name != "inRange" {
				value.Lsh(value, 512)
			}
			if name == "underflow" {
				value.Neg(value)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ConstrainIntegerToBounds(value, min, max)
			}
		})
	}
}
