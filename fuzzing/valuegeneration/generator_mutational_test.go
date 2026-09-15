package valuegeneration

import (
	"math/big"
	"math/rand"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// getTestMutationalValueGeneratorForArrays creates a MutationalValueGenerator with array structure mutations always
// applied, performing between one and five mutation rounds per call.
func getTestMutationalValueGeneratorForArrays() *MutationalValueGenerator {
	config := &MutationalValueGeneratorConfig{
		MinMutationRounds:               1,
		MaxMutationRounds:               5,
		MutateArrayStructureProbability: 1.0,
		RandomValueGeneratorConfig: &RandomValueGeneratorConfig{
			GenerateRandomArrayMinSize:  0,
			GenerateRandomArrayMaxSize:  100,
			GenerateRandomBytesMinSize:  0,
			GenerateRandomBytesMaxSize:  100,
			GenerateRandomStringMinSize: 0,
			GenerateRandomStringMaxSize: 100,
		},
	}
	return NewMutationalValueGenerator(config, NewValueSet(), rand.New(rand.NewSource(time.Now().UnixNano())))
}

// sortedBigIntValues converts the provided values to a sorted slice of big integers. The caller must ensure every
// element is a *big.Int.
func sortedBigIntValues(values []any) []*big.Int {
	result := make([]*big.Int, 0, len(values))
	for _, v := range values {
		result = append(result, v.(*big.Int))
	}
	slices.SortFunc(result, func(a, b *big.Int) int { return a.Cmp(b) })
	return result
}

// TestMutateArrayFixedLengthPreservesElements verifies that structure mutations never change the length of a
// fixed-size array and only ever reorder (never alter or nil out) its elements.
func TestMutateArrayFixedLengthPreservesElements(t *testing.T) {
	generator := getTestMutationalValueGeneratorForArrays()

	original := []any{big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(4), big.NewInt(5), big.NewInt(6)}
	originalSorted := sortedBigIntValues(original)

	for i := 0; i < 1000; i++ {
		mutated := generator.MutateArray(slices.Clone(original), true)

		// A fixed-size array must retain its exact length.
		assert.Equal(t, len(original), len(mutated), "fixed-size array length changed")

		// Structure mutations must only reorder elements, so the multiset of values must be preserved.
		assert.True(t, slices.EqualFunc(originalSorted, sortedBigIntValues(mutated), func(a, b *big.Int) bool {
			return a.Cmp(b) == 0
		}), "fixed-size array elements were altered: %v", mutated)
	}
}

// TestMutateArrayDynamicLengthMutations verifies that structure mutations on dynamic-sized arrays keep every
// non-nil element drawn from the original values (inserted placeholders are nil) while allowing the length to both
// grow and shrink across many mutation rounds.
func TestMutateArrayDynamicLengthMutations(t *testing.T) {
	generator := getTestMutationalValueGeneratorForArrays()

	original := []any{big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(4)}
	originalSorted := sortedBigIntValues(original)
	maxPossibleLength := len(original) + 5 // insert may add at most one element per mutation round

	lengthChanged := false
	for i := 0; i < 1000; i++ {
		mutated := generator.MutateArray(slices.Clone(original), false)

		// The length may shrink (delete) but can never exceed the original length plus the mutation rounds.
		assert.LessOrEqual(t, len(mutated), maxPossibleLength, "dynamic array grew beyond the possible length")

		// Every non-nil element must be one of the original values, respecting multiplicity: two-pointer scan over
		// the sorted values ensures the mutated non-nil multiset is a subset of the original multiset.
		remaining := sortedBigIntValues(nonNilElements(mutated))
		assert.True(t, isSubMultiset(remaining, originalSorted), "dynamic array contained a foreign element: %v", mutated)

		if len(mutated) != len(original) {
			lengthChanged = true
		}
	}

	// With insert and delete applicable every round, the length must have changed at least once across 1000 calls.
	assert.True(t, lengthChanged, "dynamic array length never changed across 1000 mutation rounds")
}

// nonNilElements returns the elements of the provided slice which are not nil.
func nonNilElements(values []any) []any {
	result := make([]any, 0, len(values))
	for _, v := range values {
		if v != nil {
			result = append(result, v)
		}
	}
	return result
}

// isSubMultiset reports whether the sorted slice a is a sub-multiset of the sorted slice b, comparing big integers
// by value.
func isSubMultiset(a, b []*big.Int) bool {
	i := 0
	for _, v := range a {
		for i < len(b) && b[i].Cmp(v) < 0 {
			i++
		}
		if i >= len(b) || b[i].Cmp(v) != 0 {
			return false
		}
		i++
	}
	return true
}
