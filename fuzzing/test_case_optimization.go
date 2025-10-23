package fuzzing

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/crytic/medusa-geth/accounts/abi"
	"github.com/crytic/medusa/fuzzing/calls"
	"github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/fuzzing/executiontracer"
	"github.com/crytic/medusa/logging"
	"github.com/crytic/medusa/logging/colors"
)

// OptimizationTestCase describes a test being run by a OptimizationTestCaseProvider.
type OptimizationTestCase struct {
	// status describes the status of the test case
	status TestCaseStatus
	// targetContract describes the target contract where the test case was found
	targetContract *contracts.Contract
	// targetMethod describes the target method for the test case
	targetMethod abi.Method
	// callSequence describes the call sequence that maximized the value
	callSequence *calls.CallSequence
	// shrinkCallSequenceRequest is the shrink request that will be executed to identify the optimal call sequence
	// that maximizes the value
	shrinkCallSequenceRequest *ShrinkCallSequenceRequest
	// value is used to store the maximum value returned by the test method
	value *big.Int
	// optimizationTestTrace describes the execution trace when running the callSequence
	optimizationTestTrace *executiontracer.ExecutionTrace
}

// Status describes the TestCaseStatus used to define the current state of the test.
func (t *OptimizationTestCase) Status() TestCaseStatus {
	return t.status
}

// CallSequence describes the calls.CallSequence of calls sent to the EVM which resulted in this TestCase result.
// This should be nil if the result is not related to the CallSequence.
func (t *OptimizationTestCase) CallSequence() *calls.CallSequence {
	return t.callSequence
}

// Name describes the name of the test case.
func (t *OptimizationTestCase) Name() string {
	return fmt.Sprintf("Optimization Test: %s.%s", t.targetContract.Name(), t.targetMethod.Sig)
}

// LogMessage obtains a buffer that represents the result of the OptimizationTestCase. This buffer can be passed to a logger for
// console or file logging.
func (t *OptimizationTestCase) LogMessage() *logging.LogBuffer {
	buffer := logging.NewLogBuffer()

	// If the test case never started, just log the status and name of the test case
	if t.Status() == TestCaseStatusNotStarted {
		buffer.Append(colors.GreenBold, fmt.Sprintf("[%s] ", t.Status()), colors.Bold, t.Name(), colors.Reset, "\n")
		return buffer
	}

	// We are now guaranteed to handle only test cases in the running state
	// If we weren't able to find a value greater than the minimum, the test case has failed
	minInt, _ := new(big.Int).SetString(MIN_INT, 16)
	if t.Value().Cmp(minInt) == 0 {
		t.status = TestCaseStatusFailed
	} else {
		t.status = TestCaseStatusPassed
	}
	buffer.Append(colors.GreenBold, fmt.Sprintf("[%s] ", t.Status()), colors.Bold, t.Name(), colors.Reset, "\n")

	// Notify the user we failed to find anything
	if t.status == TestCaseStatusFailed {
		buffer.Append(fmt.Sprintf("Test for method \"%s.%s\" failed to identify a value greater than the minimum"+
			" value of an int256: ", t.targetContract.Name(), t.targetMethod.Sig))
		// We do not have a call sequence or execution trace for this test, so return early
		return buffer
	}

	// We are guaranteed to now handle only successful test cases
	buffer.Append(fmt.Sprintf("Test for method \"%s.%s\" resulted in the maximum value: ", t.targetContract.Name(), t.targetMethod.Sig))
	buffer.Append(colors.Bold, t.value, colors.Reset, "\n")
	buffer.Append(colors.Bold, "[Call Sequence]", colors.Reset, "\n")
	buffer.Append(t.CallSequence().Log().Elements()...)

	// If an execution trace is attached then add it to the message
	if t.optimizationTestTrace != nil {
		buffer.Append(colors.Bold, "[Optimization Test Execution Trace]", colors.Reset, "\n")
		buffer.Append(t.optimizationTestTrace.Log().Elements()...)
	}
	return buffer
}

// Message obtains a text-based printable message which describes the result of the OptimizationTestCase.
func (t *OptimizationTestCase) Message() string {
	// Internally, we just call log message and convert it to a string. This can be useful for 3rd party apps
	return t.LogMessage().String()
}

// ID obtains a unique identifier for a test result.
func (t *OptimizationTestCase) ID() string {
	return strings.Replace(fmt.Sprintf("OPTIMIZATION-%s-%s", t.targetContract.Name(), t.targetMethod.Sig), "_", "-", -1)
}

// Value obtains the maximum value returned by the test method found till now
func (t *OptimizationTestCase) Value() *big.Int {
	return t.value
}
