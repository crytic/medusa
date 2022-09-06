package fuzzing

import "github.com/trailofbits/medusa/fuzzing/types"

const (
	TestCaseStatusNotStarted string = "NOT STARTED"
	TestCaseStatusRunning           = "RUNNING"
	TestCaseStatusPassed            = "PASSED"
	TestCaseStatusFailed            = "FAILED"
)

// TestCase describes a test being run by a TestCaseProvider.
type TestCase interface {
	// Status describes the TestCaseStatus enum option used to define the current state of the test.
	Status() string

	// CallSequence describes the types.CallSequence of calls sent to the EVM which resulted in this TestCase.
	CallSequence() types.CallSequence

	// Name describes the name of the test case.
	Name() string

	// Message obtains a text-based printable message which describes the test result.
	Message() string

	// ID obtains a unique identifier for a test result. If the same test fails, this ID should match for both
	// TestResult instances (even if the CallSequence differs or has not been shrunk).
	ID() string
}
