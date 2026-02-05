package tui

import (
	"github.com/crytic/medusa/fuzzing"
	"github.com/crytic/medusa/fuzzing/corpus"
)

// FuzzerDataProvider abstracts fuzzer data access for the TUI.
// This interface defines the minimal set of methods the TUI needs to display fuzzer state.
// The fuzzing.Fuzzer type already implements this interface.
type FuzzerDataProvider interface {
	// Lifecycle methods
	IsStopped() bool
	Stop()

	// Data access methods
	Metrics() *fuzzing.FuzzerMetrics
	Corpus() *corpus.Corpus
	Workers() []*fuzzing.FuzzerWorker
	TestCasesWithStatus(status fuzzing.TestCaseStatus) []fuzzing.TestCase
}
