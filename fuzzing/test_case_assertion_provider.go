package fuzzing

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/trailofbits/medusa/fuzzing/types"
	"math/big"
	"sync"
)

// AssertionTestCaseProvider is am AssertionTestCase provider which spawns test cases for every contract method and
// ensures that none of them result in a failed assertion (e.g. use of the solidity `assert(...)` statement, or special
// events indicating a failed assertion).
type AssertionTestCaseProvider struct {
	// fuzzer describes the Fuzzer which this provider is attached to.
	fuzzer *Fuzzer

	// testCases is a map of contract-method IDs to property test cases.GetContractMethodID
	testCases map[types.ContractMethodID]*AssertionTestCase

	// testCasesLock is used for thread-synchronization when updating testCases
	testCasesLock sync.Mutex
}

// attachAssertionTestCaseProvider attaches a new AssertionTestCaseProvider to the Fuzzer and returns it.
func attachAssertionTestCaseProvider(fuzzer *Fuzzer) *AssertionTestCaseProvider {
	// Create a test case provider
	t := &AssertionTestCaseProvider{
		fuzzer: fuzzer,
	}

	// Subscribe the provider to relevant events the fuzzer emits.
	fuzzer.OnStartingEventEmitter.Subscribe(t.onFuzzerStarting)
	fuzzer.OnStoppingEventEmitter.Subscribe(t.onFuzzerStopping)
	fuzzer.OnWorkerCreatedEventEmitter.Subscribe(t.onWorkerCreated)

	// Add the provider's call sequence test function to the fuzzer.
	fuzzer.CallSequenceTestFunctions = append(fuzzer.CallSequenceTestFunctions, t.callSequencePostCallTest)
	return t
}

// isTestableMethod checks whether the method is configured by the attached fuzzer to be a target of assertion testing.
// Returns true if this target should be tested, false otherwise.
func (t *AssertionTestCaseProvider) isTestableMethod(method abi.Method) bool {
	// Only test constant methods (pure/view) if we are configured to.
	return !method.IsConstant() || t.fuzzer.config.Fuzzing.Testing.AssertionTesting.TestViewMethods
}

// isAssertionVMError indicates whether a provided error returned from the EVM is an INVALID opcode error, indicative
// of an assertion failure.
func (t *AssertionTestCaseProvider) isAssertionVMError(err error) bool {
	// See if the error can be cased to an invalid opcode error.
	_, hitInvalidOpcode := err.(*vm.ErrInvalidOpCode)
	return hitInvalidOpcode
}

// checkAssertionFailures checks the results of the last call for assertion failures.
// Returns the method ID, a boolean indicating if an assertion test failed, or an error if one occurs.
func (t *AssertionTestCaseProvider) checkAssertionFailures(worker *FuzzerWorker, callSequence types.CallSequence) (*types.ContractMethodID, bool, error) {
	// If we have an empty call sequence, we cannot have an assertion failure
	if len(callSequence) == 0 {
		return nil, false, nil
	}

	// Obtain the contract and method from the last call made in our sequence
	lastCall := callSequence[len(callSequence)-1]
	lastCallMethod, err := lastCall.Method()
	if err != nil {
		return nil, false, err
	}
	methodId := types.GetContractMethodID(lastCall.Contract(), lastCallMethod)

	// Obtain the last block
	lastBlock := worker.chain.Head()

	// Obtain our transaction index in the block
	//  TODO: When we support multiple calls per block in the fuzzer, replace this hardcoded index.
	lastCallResults := lastBlock.MessageResults()[0]

	// Check if we encountered an assertion error.
	encounteredAssertionVMError := t.isAssertionVMError(lastCallResults.ExecutionResult.Err)

	return &methodId, encounteredAssertionVMError, nil
}

// checkPropertyTestFailed executes a given property test method to see if it returns a failed status. This is used to
// facilitate testing of property test methods after every call the Fuzzer makes when testing call sequences.
func (t *AssertionTestCaseProvider) checkPropertyTestFailed(worker *FuzzerWorker, propertyTestMethod *types.DeployedContractMethod) (bool, error) {
	// Generate our ABI input data for the call. In this case, property test methods take no arguments, so the
	// variadic argument list here is empty.
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

// onFuzzerStarting is the event handler triggered when the Fuzzer is starting a fuzzing campaign. It creates test cases
// in a "not started" state for every property test method discovered in the contract definitions known to the Fuzzer.
func (t *AssertionTestCaseProvider) onFuzzerStarting(event OnFuzzerStarting) error {
	// Reset our state
	t.testCases = make(map[types.ContractMethodID]*AssertionTestCase)

	// Create a test case for every test method.
	for _, contract := range t.fuzzer.ContractDefinitions() {
		for _, method := range contract.CompiledContract().Abi.Methods {
			if t.isTestableMethod(method) {
				// Create local variables to avoid pointer types in the loop being overridden.
				contract := contract
				method := method

				// Create our test case
				propertyTestCase := &AssertionTestCase{
					status:         TestCaseStatusNotStarted,
					targetContract: &contract,
					targetMethod:   method,
					callSequence:   nil,
				}

				// Add to our test cases and register them with the fuzzer
				methodId := types.GetContractMethodID(&contract, &method)
				t.testCases[methodId] = propertyTestCase
				t.fuzzer.RegisterTestCase(propertyTestCase)
			}
		}
	}
	return nil
}

// onFuzzerStarting is the event handler triggered when the Fuzzer is stopping the fuzzing campaign and all workers
// have been destroyed. It clears state tracked for each FuzzerWorker and sets test cases in "running" states to
// "passed".
func (t *AssertionTestCaseProvider) onFuzzerStopping(event OnFuzzerStopping) error {
	// Loop through each test case and set any tests with a running status to a passed status.
	for _, testCase := range t.testCases {
		if testCase.status == TestCaseStatusRunning {
			testCase.status = TestCaseStatusPassed
		}
	}
	return nil
}

// onWorkerCreated is the event handler triggered when a FuzzerWorker is created by the Fuzzer. It ensures state tracked
// for that worker index is refreshed and subscribes to relevant worker events.
func (t *AssertionTestCaseProvider) onWorkerCreated(event OnWorkerCreated) error {
	// Subscribe to relevant worker events.
	event.Worker.OnDeployedContractAddedEventEmitter.Subscribe(t.onWorkerDeployedContractAdded)
	return nil
}

// onWorkerDeployedContractAdded is the event handler triggered when a FuzzerWorker detects a new contract deployment
// on its underlying chain. It ensures any property test methods which the deployed contract contains are tracked by the
// provider for testing. Any test cases previously made for these methods which are in a "not started" state are put
// into a "running" state, as they are now potentially reachable for testing.
func (t *AssertionTestCaseProvider) onWorkerDeployedContractAdded(event OnWorkerDeployedContractAdded) error {
	// If we don't have a contract definition, we can't run tests against the contract.
	if event.ContractDefinition == nil {
		return nil
	}

	// Loop through all methods and find ones for which we have tests
	for _, method := range event.ContractDefinition.CompiledContract().Abi.Methods {
		// Obtain an identifier for this pair
		methodId := types.GetContractMethodID(event.ContractDefinition, &method)

		// If we have a test case targeting this contract/method that has not failed, track this deployed method in
		// our map for this worker. If we have any tests in a not-started state, we can signal a running state now.
		t.testCasesLock.Lock()
		testCase, testCaseExists := t.testCases[methodId]
		t.testCasesLock.Unlock()
		if testCaseExists && testCase.Status() == TestCaseStatusNotStarted {
			testCase.status = TestCaseStatusRunning
		}
	}
	return nil
}

// callSequencePostCallTest provides is a CallSequenceTestFunc that performs post-call testing logic for the attached Fuzzer
// and any underlying FuzzerWorker. It is called after every call made in a call sequence. It checks whether property
// test invariants are upheld after each call the Fuzzer makes when testing a call sequence.
func (t *AssertionTestCaseProvider) callSequencePostCallTest(worker *FuzzerWorker, callSequence types.CallSequence) ([]ShrinkCallSequenceRequest, error) {
	// Create a list of shrink call sequence verifiers, which we populate for each failed test we want a call sequence
	// shrunk for.
	shrinkRequests := make([]ShrinkCallSequenceRequest, 0)

	// Obtain the method ID for the last call and check if it encountered assertion failures.
	methodId, testFailed, err := t.checkAssertionFailures(worker, callSequence)
	if err != nil {
		return nil, err
	}

	// Obtain the test case for this method we're targeting for assertion testing.
	t.testCasesLock.Lock()
	testCase := t.testCases[*methodId]
	t.testCasesLock.Unlock()

	// If the test case already failed, skip it
	if testCase.Status() == TestCaseStatusFailed {
		return shrinkRequests, nil
	}

	// If we failed a test, we update our state immediately. We provide a shrink verifier which will update
	// the call sequence for each shrunken sequence provided that fails the property test.
	if testFailed {
		// Create a request to shrink this call sequence.
		shrinkRequest := ShrinkCallSequenceRequest{
			VerifierFunction: func(worker *FuzzerWorker, callSequence types.CallSequence) (bool, error) {
				// Obtain the method ID for the last call and check if it encountered assertion failures.
				methodId2, testFailed2, err := t.checkAssertionFailures(worker, callSequence)
				if err != nil {
					return false, err
				}

				// If we encountered assertion failures on the same method, this shrunk sequence is satisfactory.
				return testFailed2 && *methodId == *methodId2, nil
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

	return shrinkRequests, nil
}
