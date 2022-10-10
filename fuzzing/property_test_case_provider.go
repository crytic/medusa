package fuzzing

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/trailofbits/medusa/fuzzing/types"
	"math/big"
	"strings"
	"sync"
)

// PropertyTestCaseProvider implements TestCaseProvider which reports the results of on-chain property tests.
// Property tests are represented as publicly-accessible view functions which have a name prefix specified by a
// config.FuzzingConfig. They take no input arguments and return a boolean indicating whether the test passed.
// If a call to any on-chain property test returns false, the test signals a failed status. If no failure is found
// before the fuzzing campaign ends, the test signals a passed status.
type PropertyTestCaseProvider struct {
	// testCases is a map of contract-method IDs to property test cases.GetContractMethodID
	testCases map[types.ContractMethodID]*PropertyTestCase

	// testCasesLock is used for thread-synchronization when updating testCases
	testCasesLock sync.Mutex

	// workerStates is a slice where each element stores state for a given worker index.
	workerStates []propertyTestCaseProviderWorkerState
}

// propertyTestCaseProviderWorkerState represents the state for an individual worker maintained by
// PropertyTestCaseProvider.
type propertyTestCaseProviderWorkerState struct {
	// propertyTestMethods a mapping from contract-method ID to deployed contract-method descriptors.
	// Each deployed contract-method represents a property test method to call for evaluation. Property tests
	// should be read-only (pure/view) functions which take no input parameters and return a boolean variable
	// indicating if the property test passed.
	propertyTestMethods map[types.ContractMethodID]types.DeployedContractMethod

	// propertyTestMethodsLock is used for thread-synchronization when updating propertyTestMethods
	propertyTestMethodsLock sync.Mutex
}

// NewPropertyTestCaseProvider returns a new PropertyTestCaseProvider
func NewPropertyTestCaseProvider() *PropertyTestCaseProvider {
	provider := &PropertyTestCaseProvider{}
	return provider
}

// isPropertyTest check whether the method is a property test given potential naming prefixes it must conform to
// and its underlying input/output arguments.
func (t *PropertyTestCaseProvider) isPropertyTest(method abi.Method, propertyTestPrefixes []string) bool {
	// loop through all enabled prefixes to find a match
	for _, prefix := range propertyTestPrefixes {
		if strings.HasPrefix(method.Name, prefix) {
			if len(method.Inputs) == 0 && len(method.Outputs) == 1 && method.Outputs[0].Type.T == abi.BoolTy {
				return true
			}
		}
	}
	return false
}

// checkPropertyTestFailed executes a given property test to see if it returns a failed status.
func (t *PropertyTestCaseProvider) checkPropertyTestFailed(worker *FuzzerWorker, propertyTestMethod *types.DeployedContractMethod) (bool, error) {
	// Generate our ABI input data for the call (just the method ID, no args)
	data, err := propertyTestMethod.Contract.CompiledContract().Abi.Pack(propertyTestMethod.Method.Name)
	if err != nil {
		return false, err
	}

	// Call the underlying contract
	// TODO: Determine if we should use `Senders[0]` or have a separate funded account for the assertions.
	value := big.NewInt(0)
	msg := worker.Chain().CreateMessage(worker.Fuzzer().senders[0], &propertyTestMethod.Address, value, data)
	res, err := worker.Chain().CallContract(msg)
	if err != nil {
		return false, fmt.Errorf("failed to call property test method: %v", err)
	}

	// If our property test method call failed, we flag a failed test.
	if res.Failed() {
		return true, nil
	}

	// Decode our ABI outputs
	retVals, err := propertyTestMethod.Method.Outputs.Unpack(res.Return())
	if err != nil {
		return false, fmt.Errorf("failed to decode property test method return value: %v", err)
	}

	// We should have one return value.
	if len(retVals) != 1 {
		return false, fmt.Errorf("detected an unexpected number of return values from property test '%s'", propertyTestMethod.Method.Name)
	}

	// The one return value should be a bool
	propertyTestMethodPassed, ok := retVals[0].(bool)
	if !ok {
		return false, fmt.Errorf("failed to parse property test method success status from return value '%s'", propertyTestMethod.Method.Name)
	}

	// Return our status from our property test method
	return !propertyTestMethodPassed, nil
}

// OnFuzzerStarting is called by the fuzzing.Fuzzer upon the start of a fuzzing campaign. Any previously recorded
// TestCase should be cleared from the provider and state should be reset.
func (t *PropertyTestCaseProvider) OnFuzzerStarting(fuzzer *Fuzzer) error {
	// Reset our state
	t.testCases = make(map[types.ContractMethodID]*PropertyTestCase)
	t.workerStates = make([]propertyTestCaseProviderWorkerState, fuzzer.Config().Fuzzing.Workers)

	// Create a test case for every property test method.
	for _, contract := range fuzzer.Contracts() {
		for _, method := range contract.CompiledContract().Abi.Methods {
			if t.isPropertyTest(method, fuzzer.Config().Fuzzing.Testing.PropertyTesting.TestPrefixes) {
				// Create local variables to avoid pointer types in the loop being overridden.
				contract := contract
				method := method

				// Create our property test case
				propertyTestCase := &PropertyTestCase{
					status:         TestCaseStatusNotStarted,
					targetContract: &contract,
					targetMethod:   method,
					callSequence:   nil,
				}

				// Add to our test cases and register them with the fuzzer
				methodId := types.GetContractMethodID(&contract, &method)
				t.testCases[methodId] = propertyTestCase
				fuzzer.RegisterTestCase(propertyTestCase)
			}
		}
	}
	return nil
}

// OnFuzzerStopping is called when a fuzzing.Fuzzer's campaign is being stopped. Any TestCase which is still in a
// running state should be updated during this step and put into a finalized state. This is guaranteed to be called
// after all workers have been stopped.
func (t *PropertyTestCaseProvider) OnFuzzerStopping(fuzzer *Fuzzer) error {
	// Clear our property test methods
	t.workerStates = nil

	// Loop through each test case and set any tests with a running status to a passed status.
	for _, testCase := range t.testCases {
		if testCase.status == TestCaseStatusRunning {
			testCase.status = TestCaseStatusPassed
		}
	}
	return nil
}

// OnWorkerCreated is called when a new fuzzing.FuzzerWorker is created by the fuzzing.Fuzzer.
func (t *PropertyTestCaseProvider) OnWorkerCreated(worker *FuzzerWorker) error {
	// Create a new state for this worker.
	t.workerStates[worker.WorkerIndex()] = propertyTestCaseProviderWorkerState{
		propertyTestMethods:     make(map[types.ContractMethodID]types.DeployedContractMethod),
		propertyTestMethodsLock: sync.Mutex{},
	}
	return nil
}

// OnWorkerDestroyed is called when a previously created fuzzing.FuzzerWorker is destroyed by the fuzzing.Fuzzer.
func (t *PropertyTestCaseProvider) OnWorkerDestroyed(worker *FuzzerWorker) error {
	return nil
}

// OnWorkerDeployedContractAdded is called when a fuzzing.FuzzerWorker detects a newly deployed contract in the
// underlying Chain. If the  contract could be matched to a definition registered with the fuzzing.Fuzzer,
// it is provided as well. Otherwise, a nil contract definition is supplied.
func (t *PropertyTestCaseProvider) OnWorkerDeployedContractAdded(worker *FuzzerWorker, contractAddress common.Address, contract *types.Contract) error {
	// If we don't have a contract definition, we can't run property tests against the contract.
	if contract == nil {
		return nil
	}

	// Loop through all methods and find ones for which we have tests
	for _, method := range contract.CompiledContract().Abi.Methods {
		// Obtain an identifier for this pair
		methodId := types.GetContractMethodID(contract, &method)

		// If we have a test case targeting this contract/method that has not failed, track this deployed method in
		// our map for this worker. If we have any tests in a not-started state, we can signal a running state now.
		t.testCasesLock.Lock()
		propertyTestCase, propertyTestCaseExists := t.testCases[methodId]
		t.testCasesLock.Unlock()

		if propertyTestCaseExists {
			if propertyTestCase.Status() == TestCaseStatusNotStarted {
				propertyTestCase.status = TestCaseStatusRunning
			}
			if propertyTestCase.Status() != TestCaseStatusFailed {
				// Create our property test method reference.
				workerState := &t.workerStates[worker.WorkerIndex()]
				workerState.propertyTestMethodsLock.Lock()
				workerState.propertyTestMethods[methodId] = types.DeployedContractMethod{
					Address:  contractAddress,
					Contract: contract,
					Method:   method,
				}
				workerState.propertyTestMethodsLock.Unlock()
			}
		}
	}
	return nil
}

// OnWorkerDeployedContractDeleted is called when a fuzzing.FuzzerWorker detects a previously reported deployed
// contract that no longer exists in the underlying Chain.
func (t *PropertyTestCaseProvider) OnWorkerDeployedContractDeleted(worker *FuzzerWorker, contractAddress common.Address, contract *types.Contract) error {
	// If we don't have a contract definition, there's nothing to do.
	if contract == nil {
		return nil
	}

	// Loop through all methods and find ones for which we have tests
	for _, method := range contract.CompiledContract().Abi.Methods {
		// Obtain an identifier for this pair
		methodId := types.GetContractMethodID(contract, &method)

		// If this identifier is in our test cases map, then we remove it from our property test method lookup for
		// this worker index.
		t.testCasesLock.Lock()
		_, isPropertyTestMethod := t.testCases[methodId]
		t.testCasesLock.Unlock()

		if isPropertyTestMethod {
			// Delete our property test method reference.
			workerState := &t.workerStates[worker.WorkerIndex()]
			workerState.propertyTestMethodsLock.Lock()
			delete(workerState.propertyTestMethods, methodId)
			workerState.propertyTestMethodsLock.Unlock()
		}
	}
	return nil
}

// OnWorkerCallSequenceTesting is called before a fuzzing.FuzzerWorker generates and tests a new call sequence.
func (t *PropertyTestCaseProvider) OnWorkerCallSequenceTesting(worker *FuzzerWorker) error {
	return nil
}

// OnWorkerCallSequenceTested is called after a fuzzing.FuzzerWorker generates and tests a new call sequence.
func (t *PropertyTestCaseProvider) OnWorkerCallSequenceTested(worker *FuzzerWorker) error {
	return nil
}

// OnWorkerCallSequenceCallTested is called after a fuzzing.FuzzerWorker sends another call in a types.CallSequence
// during a fuzzing campaign. It returns a ShrinkCallSequenceRequest set, which represents a set of requests for
// shrunken call sequences alongside verifiers to guide the shrinking process. This signals to the FuzzerWorker
// that this current call sequence was interesting, and it should stop building on it and find a shrunken
// sequence that satisfies the conditions specified by the ShrinkCallSequenceRequest, before generating
// entirely new call sequences. A TestCaseProvider provider should not unnecessarily make shrink requests
// as this will cancel the current call sequence being further built upon.
func (t *PropertyTestCaseProvider) OnWorkerCallSequenceCallTested(worker *FuzzerWorker, callSequence types.CallSequence) ([]ShrinkCallSequenceRequest, error) {
	// Create a list of shrink call sequence verifiers, which we populate for each failed property test we want a call
	// sequence shrunk for.
	shrinkRequests := make([]ShrinkCallSequenceRequest, 0)

	// Obtain the property test methods for this worker
	workerState := &t.workerStates[worker.WorkerIndex()]

	// Loop through all property test methods and test them.
	for propertyTestMethodId, workerPropertyTestMethod := range workerState.propertyTestMethods {
		// Obtain the test case for this property test method
		t.testCasesLock.Lock()
		testCase := t.testCases[propertyTestMethodId]
		t.testCasesLock.Unlock()

		// If the test case already failed, skip it
		if testCase.Status() == TestCaseStatusFailed {
			continue
		}

		// Test our property test method (create a local copy to avoid loop overwriting the method)
		workerPropertyTestMethod := workerPropertyTestMethod
		failedPropertyTest, err := t.checkPropertyTestFailed(worker, &workerPropertyTestMethod)
		if err != nil {
			return nil, err
		}

		// If we failed a test, we update our state immediately. We provide a shrink verifier which will update
		// the call sequence for each shrunken sequence provided that fails the property test.
		if failedPropertyTest {
			// Create a request to shrink this call sequence.
			shrinkRequest := ShrinkCallSequenceRequest{
				VerifierFunction: func(worker *FuzzerWorker, callSequence types.CallSequence) (bool, error) {
					// First verify the contract to property test is still deployed to call upon.
					_, propertyTestContractDeployed := worker.deployedContracts[workerPropertyTestMethod.Address]
					if !propertyTestContractDeployed {
						// If the contract isn't available, this shrunk sequence likely messed up deployment, so we
						// report it as an invalid solution.
						return false, nil
					}

					// Then the shrink verifier simply ensures the previously failed property test fails
					// for the shrunk sequence as well.
					return t.checkPropertyTestFailed(worker, &workerPropertyTestMethod)
				},
				FinishedCallback: func(worker *FuzzerWorker, shrunkenCallSequence types.CallSequence) error {
					// When we're finished shrinking, update our test state and report it finalized.
					testCase.status = TestCaseStatusFailed
					testCase.callSequence = &shrunkenCallSequence
					worker.Fuzzer().ReportTestCaseFinished(testCase)
					return nil
				},
			}

			// Add our shrink request to our list.
			shrinkRequests = append(shrinkRequests, shrinkRequest)
		}
	}

	return shrinkRequests, nil
}
