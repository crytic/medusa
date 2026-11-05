package fuzzing

import (
	"fmt"
	"strings"
	"sync"

	"github.com/crytic/medusa-geth/accounts/abi"
	"github.com/crytic/medusa/fuzzing/calls"
	fuzzerTypes "github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/logging"
	"github.com/crytic/medusa/logging/colors"
)

// SometimesTestCase describes a test being run by a SometimesTestCaseProvider.
type SometimesTestCase struct {
	// lock is used for thread-synchronization when reading or updating test case fields.
	lock sync.Mutex
	// status describes the status of the test case
	status TestCaseStatus
	// targetContract describes the target contract where the test case was found
	targetContract *fuzzerTypes.Contract
	// targetMethod describes the target method for the test case
	targetMethod abi.Method
	// executionCount tracks the total number of times the test was executed
	executionCount uint64
	// successCount tracks the number of times the test succeeded (did not revert)
	successCount uint64
	// minSuccessRate is the minimum success rate required for the test to pass
	minSuccessRate float64
	// minExecutionCount is the minimum number of executions required before evaluation
	minExecutionCount uint64
}

// Status describes the TestCaseStatus used to define the current state of the test.
func (t *SometimesTestCase) Status() TestCaseStatus {
	return t.status
}

// CallSequence describes the types.CallSequence of calls sent to the EVM which resulted in this TestCase result.
// For sometimes tests, this is always nil as there's no single failing sequence.
func (t *SometimesTestCase) CallSequence() *calls.CallSequence {
	return nil
}

// Name describes the name of the test case.
func (t *SometimesTestCase) Name() string {
	return fmt.Sprintf("Sometimes Test: %s.%s", t.targetContract.Name(), t.targetMethod.Sig)
}

// SuccessRate returns the current success rate of the test.
func (t *SometimesTestCase) SuccessRate() float64 {
	if t.executionCount == 0 {
		return 0.0
	}
	return float64(t.successCount) / float64(t.executionCount)
}

// LogMessage obtains a buffer that represents the result of the SometimesTestCase. This buffer can be passed to a logger for
// console or file logging.
func (t *SometimesTestCase) LogMessage() *logging.LogBuffer {
	buffer := logging.NewLogBuffer()

	successRate := t.SuccessRate()

	if t.Status() == TestCaseStatusFailed {
		buffer.Append(colors.RedBold, fmt.Sprintf("[%s] ", t.Status()), colors.Bold, t.Name(), colors.Reset, "\n")
		buffer.Append(fmt.Sprintf("Test for method \"%s.%s\" failed:\n", t.targetContract.Name(), t.targetMethod.Sig))
		buffer.Append(fmt.Sprintf("  Executions: %d (minimum required: %d)\n", t.executionCount, t.minExecutionCount))
		buffer.Append(fmt.Sprintf("  Successes: %d\n", t.successCount))
		buffer.Append(fmt.Sprintf("  Success rate: %.2f%% (minimum required: %.2f%%)\n",
			successRate*100, t.minSuccessRate*100))
		buffer.Append(fmt.Sprintf("  The test should succeed at least %.2f%% of the time but only succeeded %.2f%% of the time.\n",
			t.minSuccessRate*100, successRate*100))
		return buffer
	}

	if t.Status() == TestCaseStatusPassed {
		buffer.Append(colors.GreenBold, fmt.Sprintf("[%s] ", t.Status()), colors.Bold, t.Name(), colors.Reset)
		if t.executionCount > 0 {
			buffer.Append(fmt.Sprintf(" (executions: %d, successes: %d, rate: %.2f%%)",
				t.executionCount, t.successCount, successRate*100))
		}
		return buffer
	}

	// For RUNNING or NOT_STARTED status
	buffer.Append(colors.Bold, fmt.Sprintf("[%s] ", t.Status()), t.Name(), colors.Reset)
	if t.executionCount > 0 {
		buffer.Append(fmt.Sprintf(" (executions: %d, successes: %d, rate: %.2f%%)",
			t.executionCount, t.successCount, successRate*100))
	}
	return buffer
}

// Message obtains a text-based printable message which describes the result of the SometimesTestCase.
func (t *SometimesTestCase) Message() string {
	return t.LogMessage().String()
}

// ID obtains a unique identifier for a test result.
func (t *SometimesTestCase) ID() string {
	return strings.Replace(fmt.Sprintf("SOMETIMES-%s-%s", t.targetContract.Name(), t.targetMethod.Sig), "_", "-", -1)
}
