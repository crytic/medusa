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
	status         TestCaseStatus
	targetContract *fuzzerTypes.Contract
	targetMethod   abi.Method
	callSequence   *calls.CallSequence
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

// Message obtains a buffer that represents the result of the AssertionTestCase. This Message can be passed to a logger for
// console / file logging or String() can be called on it to retrieve its string representation.
func (t *AssertionTestCase) Message() *logging.LogBuffer {
	// If the test failed, return a failure message.
	buffer := logging.NewLogBuffer()
	if t.Status() == TestCaseStatusFailed {
		buffer.Append(fmt.Sprintf("Test for method \"%s.%s\" resulted in an assertion failure after the following call sequence:\n", t.targetContract.Name(), t.targetMethod.Sig))
		buffer.Append(colors.Bold, "[Call Sequence]", colors.Reset, "\n")
		buffer.Append(t.CallSequence().Log().Args()...)
		return buffer
	}

	buffer.Append("")
	return buffer
}

// ID obtains a unique identifier for a test result.
func (t *AssertionTestCase) ID() string {
	return strings.Replace(fmt.Sprintf("ASSERTION-%s-%s", t.targetContract.Name(), t.targetMethod.Sig), "_", "-", -1)
}
