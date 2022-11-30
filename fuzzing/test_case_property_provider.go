package fuzzing

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/trailofbits/medusa/fuzzing/types"
	"math/big"
	"strings"
	"sync"
)

// PropertyTestCaseProvider is a provider for on-chain property tests.
// Property tests are represented as publicly-accessible view functions which have a name prefix specified by a
// config.FuzzingConfig. They take no input arguments and return a boolean indicating whether the test passed.
// If a call to any on-chain property test returns false, the test signals a failed status. If no failure is found
// before the fuzzing campaign ends, the test signals a passed status.
type PropertyTestCaseProvider struct {
	// fuzzer describes the Fuzzer which this provider is attached to.
	fuzzer *Fuzzer

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

// attachPropertyTestCaseProvider attaches a new PropertyTestCaseProvider to the Fuzzer and returns it.
func attachPropertyTestCaseProvider(fuzzer *Fuzzer) *PropertyTestCaseProvider {
	// Create a test case provider
	t := &PropertyTestCaseProvider{
		fuzzer: fuzzer,
	}

	// Subscribe the provider to relevant events the fuzzer emits.
	fuzzer.Events.FuzzerStarting.Subscribe(t.onFuzzerStarting)
	fuzzer.Events.FuzzerStopping.Subscribe(t.onFuzzerStopping)
	fuzzer.Events.WorkerCreated.Subscribe(t.onWorkerCreated)

	// Add the provider's call sequence test function to the fuzzer.
	fuzzer.Hooks.CallSequenceTestFuncs = append(fuzzer.Hooks.CallSequenceTestFuncs, t.callSequencePostCallTest)
	return t
}

// isPropertyTest check whether the method is a property test given potential naming prefixes it must conform to
// and its underlying input/output arguments.
func (t *PropertyTestCaseProvider) isPropertyTest(method abi.Method) bool {
	// Loop through all enabled prefixes to find a match
	for _, prefix := range t.fuzzer.Config().Fuzzing.Testing.PropertyTesting.TestPrefixes {
		if strings.HasPrefix(method.Name, prefix) {
			if len(method.Inputs) == 0 && len(method.Outputs) == 1 && method.Outputs[0].Type.T == abi.BoolTy {
				return true
			}
		}
	}
	return false
}

// checkPropertyTestFailed executes a given property test method to see if it returns a failed status. This is used to
// facilitate testing of property test methods after every call the Fuzzer makes when testing call sequences.
func (t *PropertyTestCaseProvider) checkPropertyTestFailed(worker *FuzzerWorker, propertyTestMethod *types.DeployedContractMethod) (bool, error) {
	// Generate our ABI input data for the call. In this case, property test methods take no arguments, so the
	// variadic argument list here is empty.
	data, err := propertyTestMethod.Contract.CompiledContract().Abi.Pack(propertyTestMethod.Method.Name)
	if err != nil {
		return false, err
	}

	// Call the underlying contract
	// TODO: Determine if we should use `Senders[0]` or have a separate funded account for the assertions.
	value := big.NewInt(0)
	msg := worker.Chain().CreateMessage(worker.Fuzzer().senders[0], &propertyTestMethod.Address, value, nil, nil, data)
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
func (t *PropertyTestCaseProvider) onFuzzerStarting(event FuzzerStartingEvent) error {
	// Reset our state
	t.testCases = make(map[types.ContractMethodID]*PropertyTestCase)
	t.workerStates = make([]propertyTestCaseProviderWorkerState, t.fuzzer.Config().Fuzzing.Workers)

	// Create a test case for every property test method.
	for _, contract := range t.fuzzer.ContractDefinitions() {
		for _, method := range contract.CompiledContract().Abi.Methods {
			if t.isPropertyTest(method) {
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
				t.fuzzer.RegisterTestCase(propertyTestCase)
			}
		}
	}
	return nil
}

// onFuzzerStarting is the event handler triggered when the Fuzzer is stopping the fuzzing campaign and all workers
// have been destroyed. It clears state tracked for each FuzzerWorker and sets test cases in "running" states to
// "passed".
func (t *PropertyTestCaseProvider) onFuzzerStopping(event FuzzerStoppingEvent) error {
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

// onWorkerCreated is the event handler triggered when a FuzzerWorker is created by the Fuzzer. It ensures state tracked
// for that worker index is refreshed and subscribes to relevant worker events.
func (t *PropertyTestCaseProvider) onWorkerCreated(event FuzzerWorkerCreatedEvent) error {
	// Create a new state for this worker.
	t.workerStates[event.Worker.WorkerIndex()] = propertyTestCaseProviderWorkerState{
		propertyTestMethods:     make(map[types.ContractMethodID]types.DeployedContractMethod),
		propertyTestMethodsLock: sync.Mutex{},
	}

	// Subscribe to relevant worker events.
	event.Worker.Events.ContractAdded.Subscribe(t.onWorkerDeployedContractAdded)
	event.Worker.Events.ContractDeleted.Subscribe(t.onWorkerDeployedContractDeleted)
	return nil
}

// onWorkerDeployedContractAdded is the event handler triggered when a FuzzerWorker detects a new contract deployment
// on its underlying chain. It ensures any property test methods which the deployed contract contains are tracked by the
// provider for testing. Any test cases previously made for these methods which are in a "not started" state are put
// into a "running" state, as they are now potentially reachable for testing.
func (t *PropertyTestCaseProvider) onWorkerDeployedContractAdded(event FuzzerWorkerContractAddedEvent) error {
	// If we don't have a contract definition, we can't run property tests against the contract.
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
		propertyTestCase, propertyTestCaseExists := t.testCases[methodId]
		t.testCasesLock.Unlock()

		if propertyTestCaseExists {
			if propertyTestCase.Status() == TestCaseStatusNotStarted {
				propertyTestCase.status = TestCaseStatusRunning
			}
			if propertyTestCase.Status() != TestCaseStatusFailed {
				// Create our property test method reference.
				workerState := &t.workerStates[event.Worker.WorkerIndex()]
				workerState.propertyTestMethodsLock.Lock()
				workerState.propertyTestMethods[methodId] = types.DeployedContractMethod{
					Address:  event.ContractAddress,
					Contract: event.ContractDefinition,
					Method:   method,
				}
				workerState.propertyTestMethodsLock.Unlock()
			}
		}
	}
	return nil
}

// onWorkerDeployedContractAdded is the event handler triggered when a FuzzerWorker detects that a previously deployed
// contract no longer exists on its underlying chain. It ensures any property test methods which the deployed contract
// contained are no longer tracked by the provider for testing.
func (t *PropertyTestCaseProvider) onWorkerDeployedContractDeleted(event FuzzerWorkerContractDeletedEvent) error {
	// If we don't have a contract definition, there's nothing to do.
	if event.ContractDefinition == nil {
		return nil
	}

	// Loop through all methods and find ones for which we have tests
	for _, method := range event.ContractDefinition.CompiledContract().Abi.Methods {
		// Obtain an identifier for this pair
		methodId := types.GetContractMethodID(event.ContractDefinition, &method)

		// If this identifier is in our test cases map, then we remove it from our property test method lookup for
		// this worker index.
		t.testCasesLock.Lock()
		_, isPropertyTestMethod := t.testCases[methodId]
		t.testCasesLock.Unlock()

		if isPropertyTestMethod {
			// Delete our property test method reference.
			workerState := &t.workerStates[event.Worker.WorkerIndex()]
			workerState.propertyTestMethodsLock.Lock()
			delete(workerState.propertyTestMethods, methodId)
			workerState.propertyTestMethodsLock.Unlock()
		}
	}
	return nil
}

// callSequencePostCallTest provides is a CallSequenceTestFunc that performs post-call testing logic for the attached Fuzzer
// and any underlying FuzzerWorker. It is called after every call made in a call sequence. It checks whether property
// test invariants are upheld after each call the Fuzzer makes when testing a call sequence.
func (t *PropertyTestCaseProvider) callSequencePostCallTest(worker *FuzzerWorker, callSequence types.CallSequence) ([]ShrinkCallSequenceRequest, error) {
	// Create a list of shrink call sequence verifiers, which we populate for each failed property test we want a call
	// sequence shrunk for.
	shrinkRequests := make([]ShrinkCallSequenceRequest, 0)

	// Obtain the test provider state for this worker
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
				VerifierFunction: func(worker *FuzzerWorker, shrunkenCallSequence types.CallSequence) (bool, error) {
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
