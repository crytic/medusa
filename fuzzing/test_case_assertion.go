package fuzzing

import (
	"fmt"
	"github.com/crytic/medusa/logging"
	"github.com/crytic/medusa/logging/colors"
	"strings"

	"github.com/crytic/medusa/fuzzing/calls"
	fuzzerTypes "github.com/crytic/medusa/fuzzing/contracts"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

// AssertionTestCase describes a test being run by a AssertionTestCaseProvider.
type AssertionTestCase struct {
	// status describes the status of the test case
	status TestCaseStatus
	// targetContract describes the target contract where the test case was found
	targetContract *fuzzerTypes.Contract
	// targetMethod describes the target method for the test case
	targetMethod abi.Method
	// callSequence describes the call sequence that broke the assertion
	callSequence *calls.CallSequence
}

// Status describes the TestCaseStatus used to define the current state of the test.
func (t *AssertionTestCase) Status() TestCaseStatus {
	return t.status
}

// CallSequence describes the types.CallSequence of calls sent to the EVM which resulted in this TestCase result.
// This should be nil if the result is not related to the CallSequence.
func (t *AssertionTestCase) CallSequence() *calls.CallSequence {
	return t.callSequence
}

// Name describes the name of the test case.
func (t *AssertionTestCase) Name() string {
	return fmt.Sprintf("Assertion Test: %s.%s", t.targetContract.Name(), t.targetMethod.Sig)
}

// LogMessage obtains a buffer that represents the result of the AssertionTestCase. This buffer can be passed to a logger for
// console or file logging.
func (t *AssertionTestCase) LogMessage() *logging.LogBuffer {
	// If the test failed, return a failure message.
	buffer := logging.NewLogBuffer()
	if t.Status() == TestCaseStatusFailed {
		buffer.Append(colors.RedBold, fmt.Sprintf("[%s] ", t.Status()), colors.Bold, t.Name(), colors.Reset, "\n")
		buffer.Append(fmt.Sprintf("Test for method \"%s.%s\" resulted in an assertion failure after the following call sequence:\n", t.targetContract.Name(), t.targetMethod.Sig))
		buffer.Append(colors.Bold, "[Call Sequence]", colors.Reset, "\n")
		buffer.Append(t.CallSequence().Log().Elements()...)
		return buffer
	}

	buffer.Append(colors.GreenBold, fmt.Sprintf("[%s] ", t.Status()), colors.Bold, t.Name(), colors.Reset)
	return buffer
}

// Message obtains a text-based printable message which describes the result of the AssertionTestCase.
func (t *AssertionTestCase) Message() string {
	// Internally, we just call log message and convert it to a string. This can be useful for 3rd party apps
	return t.LogMessage().String()
}

// ID obtains a unique identifier for a test result.
func (t *AssertionTestCase) ID() string {
	return strings.Replace(fmt.Sprintf("ASSERTION-%s-%s", t.targetContract.Name(), t.targetMethod.Sig), "_", "-", -1)
}
