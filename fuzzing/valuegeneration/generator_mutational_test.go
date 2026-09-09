package valuegeneration

import (
	"math/rand"
	"testing"
)

// TestGetMutationParamsMutationCountRange verifies that mutation counts stay within the configured bounds.
func TestGetMutationParamsMutationCountRange(t *testing.T) {
	// A non-zero lower bound exposes the bug hidden by the default minimum of zero.
	testCases := []struct {
		name string
		min  int
		max  int
	}{
		{name: "fixed non-zero range", min: 3, max: 3},
		{name: "bounded non-zero range", min: 3, max: 5},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			generator := NewMutationalValueGenerator(&MutationalValueGeneratorConfig{
				MinMutationRounds:          testCase.min,
				MaxMutationRounds:          testCase.max,
				RandomValueGeneratorConfig: &RandomValueGeneratorConfig{},
			}, NewValueSet(), rand.New(rand.NewSource(0)))

			// Sample the deterministic sequence enough times to catch an out-of-range value.
			for range 100 {
				_, mutationCount := generator.getMutationParams(1)
				if mutationCount < testCase.min || mutationCount > testCase.max {
					t.Fatalf("mutation count %d is outside [%d, %d]", mutationCount, testCase.min, testCase.max)
				}
			}
		})
	}
}
