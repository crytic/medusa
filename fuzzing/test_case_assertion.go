package fuzzing

import (
	"fmt"
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

// Message obtains a text-based printable message which describes the test result.
func (t *AssertionTestCase) Message() string {
	// If the test failed, return a failure message.
	if t.Status() == TestCaseStatusFailed {
		return fmt.Sprintf(
			"Test for method \"%s.%s\" failed after the following call sequence resulted in an assertion:\n%s",
			t.targetContract.Name(),
			t.targetMethod.Sig,
			t.CallSequence().String(),
		)
	}
	return ""
}

// ID obtains a unique identifier for a test result.
func (t *AssertionTestCase) ID() string {
	return strings.Replace(fmt.Sprintf("ASSERTION-%s-%s", t.targetContract.Name(), t.targetMethod.Sig), "_", "-", -1)
}
