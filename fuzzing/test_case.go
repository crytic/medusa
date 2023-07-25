package fuzzing

import (
	"github.com/crytic/medusa/fuzzing/calls"
	"github.com/crytic/medusa/logging"
)

// TestCaseStatus defines the status of a TestCase as a string-represented enum.
type TestCaseStatus string

const (
	// TestCaseStatusNotStarted describes a test status where conditions have not yet been tested.
	TestCaseStatusNotStarted TestCaseStatus = "NOT STARTED"
	// TestCaseStatusRunning describes a test status where conditions have been tested for but no result
	// has been reported.
	TestCaseStatusRunning TestCaseStatus = "RUNNING"
	// TestCaseStatusPassed describes a test status where testing has concluded and the test passed.
	TestCaseStatusPassed TestCaseStatus = "PASSED"
	// TestCaseStatusFailed describes a test status where testing has concluded and the test failed.
	TestCaseStatusFailed TestCaseStatus = "FAILED"
)

// TestCase describes a test which is being conducted by a test provider attached to the Fuzzer.
type TestCase interface {
	// Status describes the TestCaseStatus used to define the current state of the test.
	Status() TestCaseStatus

	// CallSequence describes the types.CallSequence of calls sent to the EVM which resulted in this TestCase result.
	// This should be nil if the result is not related to the CallSequence.
	CallSequence() *calls.CallSequence

	// Name describes the name of the test case.
	Name() string

	// LogMessage obtains a logging.LogBuffer that represents the result of the TestCase. This buffer can be passed to a logger for
	// console or file logging.
	LogMessage() *logging.LogBuffer

	// Message obtains a text-based printable message which describes the result of the AssertionTestCase.
	Message() string

	// ID obtains a unique identifier for a test result. If the same test fails, this ID should match for both
	// TestResult instances (even if the CallSequence differs or has not been shrunk).
	ID() string
}
