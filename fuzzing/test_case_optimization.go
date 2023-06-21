package fuzzing

import (
	"fmt"
	"github.com/crytic/medusa/fuzzing/calls"
	"github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/fuzzing/executiontracer"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"math/big"
	"strings"
	"sync"
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
	// value is used to store the maximum value returned by the test method
	value *big.Int
	// valueLock is used for thread-synchronization when updating the value
	valueLock sync.Mutex
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

// Message obtains a text-based printable message which describes the test result.
func (t *OptimizationTestCase) Message() string {
	// We print final value in case the test case passed for optimization test
	if t.Status() == TestCaseStatusPassed {
		msg := fmt.Sprintf(
			"Optimization test \"%s.%s\" resulted in the maximum value: %s with the following sequence:\n%s",
			t.targetContract.Name(),
			t.targetMethod.Sig,
			t.value,
			t.CallSequence().String(),
		)
		// If an execution trace is attached then add it to the message
		if t.optimizationTestTrace != nil {
			// TODO: Improve formatting in logging PR
			msg += fmt.Sprintf("\nOptimization test execution trace:\n%s", t.optimizationTestTrace.String())
		}
		return msg
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
