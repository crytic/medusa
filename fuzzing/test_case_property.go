package fuzzing

import (
	"fmt"
	"github.com/crytic/medusa/fuzzing/calls"
	fuzzerTypes "github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/fuzzing/executiontracer"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"strings"
)

// PropertyTestCase describes a test being run by a PropertyTestCaseProvider.
type PropertyTestCase struct {
	// status describes the status of the test case
	status TestCaseStatus
	// targetContract describes the target contract where the test case was found
	targetContract *fuzzerTypes.Contract
	// targetMethod describes the target method for the test case
	targetMethod abi.Method
	// callSequence describes the call sequence that broke the property
	callSequence *calls.CallSequence
	// propertyTestTrace describes the execution trace when running the callSequence
	propertyTestTrace *executiontracer.ExecutionTrace
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
		msg := fmt.Sprintf(
			"Property test \"%s.%s\" failed after the following call sequence:\n%s",
			t.targetContract.Name(),
			t.targetMethod.Sig,
			t.CallSequence().String(),
		)
		// If an execution trace is attached then add it to the message
		if t.propertyTestTrace != nil {
			// TODO: Improve formatting in logging PR
			msg += fmt.Sprintf("\nProperty test execution trace:\n%s", t.propertyTestTrace.String())
		}
		return msg
	}
	return ""
}

// ID obtains a unique identifier for a test result.
func (t *PropertyTestCase) ID() string {
	return strings.Replace(fmt.Sprintf("PROPERTY-%s-%s", t.targetContract.Name(), t.targetMethod.Sig), "_", "-", -1)
}
