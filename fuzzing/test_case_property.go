package fuzzing

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/trailofbits/medusa/fuzzing/calls"
	fuzzerTypes "github.com/trailofbits/medusa/fuzzing/contracts"
	"strings"
)

// PropertyTestCase describes a test being run by a PropertyTestCaseProvider.
type PropertyTestCase struct {
	status         TestCaseStatus
	targetContract *fuzzerTypes.Contract
	targetMethod   abi.Method
	callSequence   *calls.CallSequence
}

// Status describes the TestCaseStatus used to define the current state of the test.
func (t *PropertyTestCase) Status() TestCaseStatus {
	return t.status
}

// CallSequence describes the types.CallSequence of calls sent to the EVM which resulted in this TestCase result.
// This should be nil if the result is not related to the CallSequence.
func (t *PropertyTestCase) CallSequence() *calls.CallSequence {
	return t.callSequence
}

// Name describes the name of the test case.
func (t *PropertyTestCase) Name() string {
	return fmt.Sprintf("Property Test: %s.%s", t.targetContract.Name(), t.targetMethod.Sig)
}

// Message obtains a text-based printable message which describes the test result.
func (t *PropertyTestCase) Message() string {
	// If the test failed, return a failure message.
	if t.Status() == TestCaseStatusFailed {
		return fmt.Sprintf(
			"Test \"%s.%s\" failed after the following call sequence:\n%s",
			t.targetContract.Name(),
			t.targetMethod.Sig,
			t.CallSequence().String(),
		)
	}
	return ""
}

// ID obtains a unique identifier for a test result.
func (t *PropertyTestCase) ID() string {
	return strings.Replace(fmt.Sprintf("PROPERTY-%s-%s", t.targetContract.Name(), t.targetMethod.Sig), "_", "-", -1)
}
