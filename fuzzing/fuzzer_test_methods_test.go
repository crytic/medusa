package fuzzing

import (
	"testing"

	"github.com/crytic/medusa/compilation"
	"github.com/crytic/medusa/events"
	"github.com/crytic/medusa/fuzzing/config"
	"github.com/stretchr/testify/assert"
)

// fuzzerTestContext holds the current Fuzzer test context and can be used for post-execution checks
// such as testing that events were properly emitted
type fuzzerTestContext struct {
	// t describes the standard testing state.
	t *testing.T

	// fuzzer refers to the Fuzzer instance spun up for the provided fuzzer test.
	fuzzer *Fuzzer

	// postExecutionChecks refers to methods that should be executed after a test method has concluded executing.
	// This is used by the Fuzzer testing API to perform some action immediately and expect a result later.
	postExecutionChecks []func(*testing.T, *fuzzerTestContext)

	// eventCounter counts events for a given type. This is used to assert events were emitted.
	eventCounter map[string]int
}

// executeFuzzerTestMethodInternal takes a provided project configuration, creates a fuzzerTestContext, and executes
// the provided test method for that context.
func executeFuzzerTestMethodInternal(t *testing.T, config *config.ProjectConfig, method func(fc *fuzzerTestContext)) {
	// Create a fuzzer instance with the provided config
	fuzzer, err := NewFuzzer(*config)
	assert.NoError(t, err)

	// Create new fuzzing context
	f := &fuzzerTestContext{
		t:                   t,
		fuzzer:              fuzzer,
		postExecutionChecks: make([]func(*testing.T, *fuzzerTestContext), 0),
		eventCounter:        make(map[string]int),
	}

	// Run the test method
	method(f)

	// Call post-execution checks
	for _, fxn := range f.postExecutionChecks {
		fxn(t, f)
	}
}

// getFuzzerTestingProjectConfig creates a default project configuration used for testing the Fuzzer.
func getFuzzerTestingProjectConfig(t *testing.T, compilationConfig *compilation.CompilationConfig) *config.ProjectConfig {
	projectConfig, err := config.GetDefaultProjectConfig("")
	assert.NoError(t, err)
	projectConfig.Compilation = compilationConfig
	projectConfig.Fuzzing.Workers = 3
	projectConfig.Fuzzing.WorkerResetLimit = 50
	projectConfig.Fuzzing.Timeout = 0
	projectConfig.Fuzzing.TestLimit = 1_500_000
	projectConfig.Fuzzing.CallSequenceLength = 100
	projectConfig.Fuzzing.Testing.StopOnFailedContractMatching = true
	projectConfig.Fuzzing.Testing.TestAllContracts = false
	return projectConfig
}

// assertFailedTestsExpected will check to see whether there are any failed tests. If `expectFailure` is false, then
// there should be no failed tests
func assertFailedTestsExpected(f *fuzzerTestContext, expectFailure bool) {
	// Ensure we captured a failed test, if expected
	failedTestCount := len(f.fuzzer.TestCasesWithStatus(TestCaseStatusFailed))
	if expectFailure {
		assert.Greater(f.t, failedTestCount, 0, "Fuzz test could not be solved before reaching limits")
	} else {
		assert.EqualValues(f.t, 0, failedTestCount, "Fuzz test failed when it should not have")
	}
}

// assertCorpusCallSequencesCollected will check to see whether we captured coverage-increasing call sequences in the
// corpus. It asserts that the actual result matches the provided expected result.
func assertCorpusCallSequencesCollected(f *fuzzerTestContext, expectCallSequences bool) {
	// Obtain our count of mutable (often representing just non-reverted coverage increasing) sequences.
	callSequenceCount, _ := f.fuzzer.corpus.CallSequenceEntryCount()

	// Ensure we captured some coverage-increasing call sequences.
	if expectCallSequences {
		assert.Greater(f.t, callSequenceCount, 0, "No coverage was captured")
	}

	// If we don't expect coverage-increasing call sequences, or it is not enabled, we should not get any coverage
	if !expectCallSequences || !f.fuzzer.config.Fuzzing.CoverageEnabled {
		assert.EqualValues(f.t, 0, callSequenceCount, "Coverage was captured")
	}
}

// expectEventEmitted will subscribe to some event T, update the eventCounter for that event (when the event callback is
// triggered) and then also add a post execution check to make sure that the event was captured properly.
func expectEventEmitted[T any](f *fuzzerTestContext, eventEmitter *events.EventEmitter[T]) {
	// Get the stringified event type for the mapping
	eventType := eventEmitter.EventType().String()

	// Subscribe to the event T and update the counter when the event is published
	eventEmitter.Subscribe(func(event T) error {
		f.eventCounter[eventType] += 1
		return nil
	})

	// Add a check to make sure that event T was published at least once
	f.postExecutionChecks = append(f.postExecutionChecks, func(t *testing.T, fctx *fuzzerTestContext) {
		assert.Greater(f.t, f.eventCounter[eventType], 0, "Event was not emitted at all")
	})
}
