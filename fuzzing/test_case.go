package fuzzing

import "github.com/trailofbits/medusa/fuzzing/types"

const (
	TestCaseStatusNotStarted string = "not started"
	TestCaseStatusRunning           = "running"
	TestCaseStatusPassed            = "passed"
	TestCaseStatusFailed            = "failed"
)

// TestCase describes a test being run by a TestCaseProvider.
type TestCase interface {
	// Status describes the TestCaseStatus enum option used to define the current state of the test.
	Status() string

	// CallSequence describes the types.CallSequence of calls sent to the EVM which resulted in this TestCase.
	CallSequence() types.CallSequence

	// String obtains a text-based printable message which describes the test result.
	String() string

	// ID obtains a unique identifier for a test result. If the same test fails, this ID should match for both
	// TestResult instances (even if the CallSequence differs or has not been shrunk).
	ID() string
}
