package fuzzing

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	fuzzerTypes "github.com/trailofbits/medusa/fuzzing/types"
)

// PropertyTestCase describes a test being run by a PropertyTestCaseProvider.
type PropertyTestCase struct {
	status         string
	targetContract *fuzzerTypes.Contract
	targetMethod   abi.Method
	callSequence   fuzzerTypes.CallSequence
}

// Status describes the TestCaseStatus enum option used to define the current state of the test.
func (t *PropertyTestCase) Status() string {
	return t.status
}

// CallSequence describes the types.CallSequence of calls sent to the EVM which resulted in this TestCase.
func (t *PropertyTestCase) CallSequence() fuzzerTypes.CallSequence {
	return t.callSequence
}

// String obtains a text-based printable message which describes the test result.
func (t *PropertyTestCase) String() string {
	// TODO
	return ""
}

// ID obtains a unique identifier for a test result. If the same test fails, this ID should match for both
// PropertyTestResult instances (even if the CallSequence differs or has not been shrunk).
func (t *PropertyTestCase) ID() string {
	return fmt.Sprintf("PROPTEST-%s-%s", t.targetContract.Name(), t.targetMethod.Sig)
}
