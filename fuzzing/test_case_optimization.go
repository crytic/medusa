package fuzzing

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	fuzzerTypes "github.com/trailofbits/medusa/fuzzing/types"
	"math/big"
	"strings"
	"sync"
)

// OptimizationTestCase describes a test being run by a OptimizationTestCaseProvider.
type OptimizationTestCase struct {
	status         TestCaseStatus
	targetContract *fuzzerTypes.Contract
	targetMethod   abi.Method
	callSequence   *fuzzerTypes.CallSequence

	// value is used to store the maximum value returned by the test method
	value *big.Int
	// valueLock is used for thread-synchronization when updating the value
	valueLock sync.Mutex
}

// Status describes the TestCaseStatus used to define the current state of the test.
func (t *OptimizationTestCase) Status() TestCaseStatus {
	return t.status
}

// CallSequence describes the types.CallSequence of calls sent to the EVM which resulted in this TestCase result.
// This should be nil if the result is not related to the CallSequence.
func (t *OptimizationTestCase) CallSequence() *fuzzerTypes.CallSequence {
	return t.callSequence
}

// Name describes the name of the test case.
func (t *OptimizationTestCase) Name() string {
	return fmt.Sprintf("Optimization Test: %s.%s", t.targetContract.Name(), t.targetMethod.Sig)
}

// Message obtains a text-based printable message which describes the test result.
func (t *OptimizationTestCase) Message() string {
	// If the test failed, return a failure message.
	if t.Status() == TestCaseStatusFailed {
		return fmt.Sprintf(
			"Test for method \"%s.%s\" failed after the following call sequence:\n%s",
			t.targetContract.Name(),
			t.targetMethod.Sig,
			t.CallSequence().String(),
		)
	}

	// We print final value in case the test case passed for optimization test
	if t.Status() == TestCaseStatusPassed {
		return fmt.Sprintf(
			"Optimization test for method \"%s.%s\" resulted in the maximum value: %s",
			t.targetContract.Name(),
			t.targetMethod.Sig,
			t.value,
		)
	}
	return ""
}

// ID obtains a unique identifier for a test result.
func (t *OptimizationTestCase) ID() string {
	return strings.Replace(fmt.Sprintf("OPTIMIZATION-%s-%s", t.targetContract.Name(), t.targetMethod.Sig), "_", "-", -1)
}

// Value obtains the maximum value returned by the test method found till now
func (t *OptimizationTestCase) Value() *big.Int {
	return t.value
}

// UpdateValue updates test case value if the new value is larger than the existing value
func (t *OptimizationTestCase) UpdateValue(newValue *big.Int) {
	t.valueLock.Lock()
	defer t.valueLock.Unlock()

	if newValue.Cmp(t.value) == 1 {
		t.value = new(big.Int).Set(newValue)
	}
}
