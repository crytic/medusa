package fuzzing

import (
	"github.com/crytic/medusa/compilation/abiutils"
	"github.com/crytic/medusa/fuzzing/calls"
	"golang.org/x/exp/slices"
	"sync"

	"github.com/crytic/medusa/fuzzing/contracts"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

// AssertionTestCaseProvider is am AssertionTestCase provider which spawns test cases for every contract method and
// ensures that none of them result in a failed assertion (e.g. use of the solidity `assert(...)` statement, or special
// events indicating a failed assertion).
type AssertionTestCaseProvider struct {
	// fuzzer describes the Fuzzer which this provider is attached to.
	fuzzer *Fuzzer

	// testCases is a map of contract-method IDs to assertion test cases.GetContractMethodID
	testCases map[contracts.ContractMethodID]*AssertionTestCase

	// testCasesLock is used for thread-synchronization when updating testCases
	testCasesLock sync.Mutex
}

// Define our ABI method

// attachAssertionTestCaseProvider attaches a new AssertionTestCaseProvider to the Fuzzer and returns it.
func attachAssertionTestCaseProvider(fuzzer *Fuzzer) *AssertionTestCaseProvider {
	// Create a test case provider
	t := &AssertionTestCaseProvider{
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

// isTestableMethod checks whether the method is configured by the attached fuzzer to be a target of assertion testing.
// Returns true if this target should be tested, false otherwise.
func (t *AssertionTestCaseProvider) isTestableMethod(method abi.Method) bool {
	// Only test constant methods (pure/view) if we are configured to.
	return !method.IsConstant() || t.fuzzer.config.Fuzzing.Testing.AssertionTesting.TestViewMethods
}

// checkAssertionFailures checks the results of the last call for assertion failures.
// Returns the method ID, a boolean indicating if an assertion test failed, or an error if one occurs.
func (t *AssertionTestCaseProvider) checkAssertionFailures(callSequence calls.CallSequence) (*contracts.ContractMethodID, bool, error) {
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
	methodId := contracts.GetContractMethodID(lastCall.Contract, lastCallMethod)

	// Check if we encountered an assertion error.
	// Try to unpack our error and return data for a panic code and verify it matches the "assert failed" panic code.
	// Solidity >0.8.0 introduced asserts failing as reverts but with special return data. But we indicate we also
	// want to be backwards compatible with older Solidity which simply hit an invalid opcode and did not actually
	// have a panic code.
	lastExecutionResult := lastCall.ChainReference.MessageResults().ExecutionResult
	panicCode := abiutils.GetSolidityPanicCode(lastExecutionResult.Err, lastExecutionResult.ReturnData, true)
	encounteredAssertionFailure := panicCode != nil && panicCode.Uint64() == abiutils.PanicCodeAssertFailed

	return &methodId, encounteredAssertionFailure, nil
}

// onFuzzerStarting is the event handler triggered when the Fuzzer is starting a fuzzing campaign. It creates test cases
// in a "not started" state for every method to test discovered in the contract definitions known to the Fuzzer.
func (t *AssertionTestCaseProvider) onFuzzerStarting(event FuzzerStartingEvent) error {
	// Reset our state
	t.testCases = make(map[contracts.ContractMethodID]*AssertionTestCase)

	// Create a test case for every test method.
	for _, contract := range t.fuzzer.ContractDefinitions() {
		// If we're not testing all contracts, verify the current contract is one we specified in our deployment order.
		if !t.fuzzer.config.Fuzzing.Testing.TestAllContracts && !slices.Contains(t.fuzzer.config.Fuzzing.DeploymentOrder, contract.Name()) {
			continue
		}

		for _, method := range contract.CompiledContract().Abi.Methods {
			// Verify this method is an assertion testable method
			if !t.isTestableMethod(method) {
				continue
			}

			// Create local variables to avoid pointer types in the loop being overridden.
			contract := contract
			method := method

			// Create our test case
			testCase := &AssertionTestCase{
				status:         TestCaseStatusNotStarted,
				targetContract: contract,
				targetMethod:   method,
				callSequence:   nil,
			}

			// Add to our test cases and register them with the fuzzer
			methodId := contracts.GetContractMethodID(contract, &method)
			t.testCases[methodId] = testCase
			t.fuzzer.RegisterTestCase(testCase)
		}
	}
	return nil
}

// onFuzzerStopping is the event handler triggered when the Fuzzer is stopping the fuzzing campaign and all workers
// have been destroyed. It clears state tracked for each FuzzerWorker and sets test cases in "running" states to
// "passed".
func (t *AssertionTestCaseProvider) onFuzzerStopping(event FuzzerStoppingEvent) error {
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
func (t *AssertionTestCaseProvider) onWorkerCreated(event FuzzerWorkerCreatedEvent) error {
	// Subscribe to relevant worker events.
	event.Worker.Events.ContractAdded.Subscribe(t.onWorkerDeployedContractAdded)
	return nil
}

// onWorkerDeployedContractAdded is the event handler triggered when a FuzzerWorker detects a new contract deployment
// on its underlying chain. It ensures any methods to test which the deployed contract contains are tracked by the
// provider for testing. Any test cases previously made for these methods which are in a "not started" state are put
// into a "running" state, as they are now potentially reachable for testing.
func (t *AssertionTestCaseProvider) onWorkerDeployedContractAdded(event FuzzerWorkerContractAddedEvent) error {
	// If we don't have a contract definition, we can't run tests against the contract.
	if event.ContractDefinition == nil {
		return nil
	}

	// Loop through all methods and find ones for which we have tests
	for _, method := range event.ContractDefinition.CompiledContract().Abi.Methods {
		// Obtain an identifier for this pair
		methodId := contracts.GetContractMethodID(event.ContractDefinition, &method)

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
// and any underlying FuzzerWorker. It is called after every call made in a call sequence. It checks whether invariants
// in methods to test are upheld after each call the Fuzzer makes when testing a call sequence.
func (t *AssertionTestCaseProvider) callSequencePostCallTest(worker *FuzzerWorker, callSequence calls.CallSequence) ([]ShrinkCallSequenceRequest, error) {
	// Create a list of shrink call sequence verifiers, which we populate for each failed test we want a call sequence
	// shrunk for.
	shrinkRequests := make([]ShrinkCallSequenceRequest, 0)

	// Obtain the method ID for the last call and check if it encountered assertion failures.
	methodId, testFailed, err := t.checkAssertionFailures(callSequence)
	if err != nil {
		return nil, err
	}

	// Obtain the test case for this method we're targeting for assertion testing.
	t.testCasesLock.Lock()
	testCase, testCaseExists := t.testCases[*methodId]
	t.testCasesLock.Unlock()

	// Verify a test case exists for this method called (if we're not assertion testing this method, stop)
	if !testCaseExists {
		return shrinkRequests, nil
	}

	// If the test case already failed, skip it
	if testCase.Status() == TestCaseStatusFailed {
		return shrinkRequests, nil
	}

	// If we failed a test, we update our state immediately. We provide a shrink verifier which will update
	// the call sequence for each shrunken sequence provided that fails the test.
	if testFailed {
		// Create a request to shrink this call sequence.
		shrinkRequest := ShrinkCallSequenceRequest{
			VerifierFunction: func(worker *FuzzerWorker, shrunkenCallSequence calls.CallSequence) (bool, error) {
				// Obtain the method ID for the last call and check if it encountered assertion failures.
				shrunkSeqMethodId, shrunkSeqTestFailed, err := t.checkAssertionFailures(shrunkenCallSequence)
				if err != nil {
					return false, err
				}

				// If we encountered assertion failures on the same method, this shrunk sequence is satisfactory.
				return shrunkSeqTestFailed && *methodId == *shrunkSeqMethodId, nil
			},
			FinishedCallback: func(worker *FuzzerWorker, shrunkenCallSequence calls.CallSequence) error {
				// When we're finished shrinking, attach an execution trace to the last call
				if len(shrunkenCallSequence) > 0 {
					err = shrunkenCallSequence[len(shrunkenCallSequence)-1].AttachExecutionTrace(worker.chain, worker.fuzzer.contractDefinitions)
					if err != nil {
						return err
					}
				}

				// Update our test state and report it finalized.
				testCase.status = TestCaseStatusFailed
				testCase.callSequence = &shrunkenCallSequence
				worker.Fuzzer().ReportTestCaseFinished(testCase)
				return nil
			},
			RecordResultInCorpus: true,
		}

		// Add our shrink request to our list.
		shrinkRequests = append(shrinkRequests, shrinkRequest)
	}

	return shrinkRequests, nil
}
