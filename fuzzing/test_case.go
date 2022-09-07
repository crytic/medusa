package fuzzing

import "github.com/trailofbits/medusa/fuzzing/types"

const (
	// TestCaseStatusNotStarted describes a test status where conditions have not yet been tested.
	TestCaseStatusNotStarted string = "NOT STARTED"
	// TestCaseStatusRunning describes a test status where conditions have been tested for but no result
	// has been reported.
	TestCaseStatusRunning = "RUNNING"
	// TestCaseStatusPassed describes a test status where testing has concluded and the test passed.
	TestCaseStatusPassed = "PASSED"
	// TestCaseStatusFailed describes a test status where testing has concluded and the test failed.
	TestCaseStatusFailed = "FAILED"
)

// TestCase describes a test being run by a TestCaseProvider.
type TestCase interface {
	// Status describes the TestCaseStatus enum option used to define the current state of the test.
	Status() string

	// CallSequence describes the types.CallSequence of calls sent to the EVM which resulted in this TestCase result.
	// This should be nil if the result is not related to the CallSequence.
	CallSequence() *types.CallSequence

	// Name describes the name of the test case.
	Name() string

	// Message obtains a text-based printable message which describes the test result.
	Message() string

	// ID obtains a unique identifier for a test result. If the same test fails, this ID should match for both
	// TestResult instances (even if the CallSequence differs or has not been shrunk).
	ID() string
}
