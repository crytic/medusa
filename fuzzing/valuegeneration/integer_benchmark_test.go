package valuegeneration

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/crytic/medusa/utils"
	"github.com/stretchr/testify/require"
)

func integerBenchmarkGenerator(size int) *MutationalValueGenerator {
	values := NewValueSet()
	for i := 0; i < size; i++ {
		values.AddInteger(big.NewInt(int64(i - size/2)))
	}
	return NewMutationalValueGenerator(&MutationalValueGeneratorConfig{
		MaxMutationRounds:          4,
		MutateIntegerProbability:   1,
		RandomValueGeneratorConfig: &RandomValueGeneratorConfig{},
	}, values, rand.New(rand.NewSource(1)))
}

func TestIntegerMutationBounds(t *testing.T) {
	t.Parallel()
	for _, signed := range []bool{true, false} {
		for _, width := range []int{8, 32, 64, 256} {
			t.Run(fmt.Sprintf("signed=%t/width=%d", signed, width), func(t *testing.T) {
				generator := integerBenchmarkGenerator(256)
				shrinker := NewShrinkingValueMutator(
					&ShrinkingValueMutatorConfig{ShrinkValueProbability: 1},
					generator.valueSet, rand.New(rand.NewSource(2)),
				)
				min, max := utils.GetIntegerConstraints(signed, width)
				input := new(big.Int).Lsh(big.NewInt(1), 300)
				original := new(big.Int).Set(input)
				for i := 0; i < 1000; i++ {
					generated := generator.GenerateInteger(signed, width)
					require.GreaterOrEqual(t, generated.Cmp(min), 0)
					require.LessOrEqual(t, generated.Cmp(max), 0)
					mutated := generator.MutateInteger(input, signed, width)
					require.GreaterOrEqual(t, mutated.Cmp(min), 0)
					require.LessOrEqual(t, mutated.Cmp(max), 0)
					shrunk := shrinker.MutateInteger(input, signed, width)
					require.GreaterOrEqual(t, shrunk.Cmp(min), 0)
					require.LessOrEqual(t, shrunk.Cmp(max), 0)
					require.Zero(t, input.Cmp(original))
				}
				for key, value := range generator.valueSet.integers {
					require.Equal(t, key, value.String(), "mutation changed a seed value")
				}
			})
		}
	}
}

// BenchmarkIntegerGeneration measures mutation-based integer generation at several seed-set sizes.
func BenchmarkIntegerGeneration(b *testing.B) {
	for _, size := range []int{8, 256, 4096} {
		b.Run(fmt.Sprintf("seeds=%d", size), func(b *testing.B) {
			generator := integerBenchmarkGenerator(size)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				generator.GenerateInteger(true, 256)
			}
		})
	}
}
