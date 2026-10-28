package fuzzing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestShouldMonitorCorpusInitialization verifies the condition which decides whether corpus initialization is worth
// monitoring. The corpus must still be initializing and must hold at least one entry. A corpus holding only
// coverage-increasing call sequences, or only test results, must still be monitored; this is a regression test for the
// condition previously requiring both kinds of entries to be present, which silently skipped monitoring (and therefore
// the corpus health log) for either of those two cases.
func TestShouldMonitorCorpusInitialization(t *testing.T) {
	tests := []struct {
		// name describes the scenario being tested.
		name string

		// initializingCorpus describes whether the corpus still has unexecuted call sequences.
		initializingCorpus bool

		// totalSequences describes the number of coverage-increasing call sequences in the corpus.
		totalSequences int

		// totalTestResults describes the number of test result call sequences in the corpus.
		totalTestResults int

		// expected describes whether initialization should be monitored.
		expected bool
	}{
		{
			name:               "only coverage-increasing call sequences",
			initializingCorpus: true,
			totalSequences:     1000,
			expected:           true,
		},
		{
			name:               "only test results",
			initializingCorpus: true,
			totalTestResults:   10,
			expected:           true,
		},
		{
			name:               "both kinds of entries",
			initializingCorpus: true,
			totalSequences:     1000,
			totalTestResults:   10,
			expected:           true,
		},
		{
			name:               "empty corpus",
			initializingCorpus: true,
			expected:           false,
		},
		{
			name:             "corpus already initialized",
			totalSequences:   1000,
			totalTestResults: 10,
			expected:         false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, shouldMonitorCorpusInitialization(test.initializingCorpus, test.totalSequences, test.totalTestResults))
		})
	}
}

// TestCalculateCorpusHealth verifies the corpus health statistics reported once corpus initialization completes.
// The reported statistics must stay consistent for a corpus which holds only coverage-increasing call sequences, only
// test results, or a mix of both. This is a regression test for the invalid entry count being derived from only the
// coverage-increasing call sequences, which reported a negative number of invalid entries whenever the valid entries
// included test results.
func TestCalculateCorpusHealth(t *testing.T) {
	tests := []struct {
		// name describes the scenario being tested.
		name string

		// totalSequences describes the number of coverage-increasing call sequences in the corpus.
		totalSequences int

		// totalTestResults describes the number of test result call sequences in the corpus.
		totalTestResults int

		// validSequences describes the number of corpus entries which replayed successfully during initialization.
		// This counter covers both kinds of corpus entries.
		validSequences uint64
	}{
		{
			name:           "only coverage-increasing call sequences",
			totalSequences: 8,
			validSequences: 8,
		},
		{
			name:             "only test results",
			totalTestResults: 5,
			validSequences:   5,
		},
		{
			name:             "both kinds of entries, all valid",
			totalSequences:   12,
			totalTestResults: 3,
			validSequences:   15,
		},
		{
			name:             "both kinds of entries, some invalid",
			totalSequences:   12,
			totalTestResults: 3,
			validSequences:   9,
		},
		{
			name: "empty corpus",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			totalEntries, validEntries, invalidEntries := calculateCorpusHealth(test.totalSequences, test.totalTestResults, test.validSequences)

			// The total must always cover both kinds of corpus entries.
			assert.Equal(t, test.totalSequences+test.totalTestResults, totalEntries)
			assert.Equal(t, int(test.validSequences), validEntries)

			// The invalid count must be derived from the total number of corpus entries rather than only the
			// coverage-increasing call sequences, and must never be reported as a negative number.
			assert.Equal(t, totalEntries-validEntries, invalidEntries)
			assert.GreaterOrEqual(t, invalidEntries, 0, "invalid entry count must not be negative")
			assert.Equal(t, totalEntries, validEntries+invalidEntries)
		})
	}
}
