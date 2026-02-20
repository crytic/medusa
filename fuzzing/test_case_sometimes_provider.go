package fuzzing

import (
	"fmt"
	"math/big"
	"slices"
	"sync"

	"github.com/crytic/medusa/fuzzing/calls"
	"github.com/crytic/medusa/fuzzing/contracts"
)

// SometimesTestCaseProvider is a provider for on-chain sometimes tests.
// Sometimes tests are represented as publicly-accessible functions which have a name prefix specified by a
// config.FuzzingConfig. They take no input arguments and should succeed (not revert) some minimum percentage
// of the time. If a sometimes test doesn't meet the minimum success rate threshold after sufficient executions,
// the test signals a failed status. If the threshold is met before the fuzzing campaign ends, the test signals
// a passed status.
type SometimesTestCaseProvider struct {
	// fuzzer describes the Fuzzer which this provider is attached to.
	fuzzer *Fuzzer

	// testCases is a map of contract-method IDs to sometimes test cases.
	testCases map[contracts.ContractMethodID]*SometimesTestCase

	// testCasesLock is used for thread-synchronization when updating testCases
	testCasesLock sync.Mutex

	// workerStates is a slice where each element stores state for a given worker index.
	workerStates []sometimesTestCaseProviderWorkerState
}

// sometimesTestCaseProviderWorkerState represents the state for an individual worker maintained by
// SometimesTestCaseProvider.
type sometimesTestCaseProviderWorkerState struct {
	// sometimesTestMethods a mapping from contract-method ID to deployed contract-method descriptors.
	// Each deployed contract-method represents a sometimes test method to call for evaluation. Sometimes tests
	// should take no input parameters and should succeed (not revert) at least some minimum percentage of executions.
	sometimesTestMethods map[contracts.ContractMethodID]contracts.DeployedContractMethod

	// sometimesTestMethodsLock is used for thread-synchronization when updating sometimesTestMethods
	sometimesTestMethodsLock sync.Mutex
}

// attachSometimesTestCaseProvider attaches a new SometimesTestCaseProvider to the Fuzzer and returns it.
func attachSometimesTestCaseProvider(fuzzer *Fuzzer) *SometimesTestCaseProvider {
	// If there are no testing prefixes, then there is no reason to attach a test case provider and subscribe to events
	if len(fuzzer.config.Fuzzing.Testing.SometimesTesting.TestPrefixes) == 0 {
		return nil
	}

	// Create a test case provider
	t := &SometimesTestCaseProvider{
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

// executeSometimesTest executes a given sometimes test method to see if it succeeds (does not revert).
// Returns a boolean indicating if the sometimes test succeeded (true) or failed/reverted (false),
// or an error if one occurred during execution.
func (t *SometimesTestCaseProvider) executeSometimesTest(worker *FuzzerWorker, sometimesTestMethod *contracts.DeployedContractMethod) (bool, error) {
	// Generate our ABI input data for the call. In this case, sometimes test methods take no arguments, so the
	// variadic argument list here is empty.
	data, err := sometimesTestMethod.Contract.CompiledContract().Abi.Pack(sometimesTestMethod.Method.Name)
	if err != nil {
		return false, err
	}

	// Create a call targeting our sometimes test method
	msg := calls.NewCallMessage(worker.Fuzzer().senders[0], &sometimesTestMethod.Address, 0, big.NewInt(0), worker.fuzzer.config.Fuzzing.TransactionGasLimit, nil, nil, nil, data)
	msg.FillFromTestChainProperties(worker.chain)

	// Execute the call.
	executionResult, err := worker.Chain().CallContract(msg.ToCoreMessage(), nil)
	if err != nil {
		return false, fmt.Errorf("failed to call sometimes test method: %v", err)
	}

	// If our sometimes test method call succeeded (didn't revert), return true
	// If it reverted, return false (this is expected behavior for sometimes tests)
	return !executionResult.Failed(), nil
}

// onFuzzerStarting is the event handler triggered when the Fuzzer is starting a fuzzing campaign. It creates test cases
// in a "not started" state for every sometimes test method discovered in the contract definitions known to the Fuzzer.
func (t *SometimesTestCaseProvider) onFuzzerStarting(event FuzzerStartingEvent) error {
	// Reset our state
	t.testCases = make(map[contracts.ContractMethodID]*SometimesTestCase)
	t.workerStates = make([]sometimesTestCaseProviderWorkerState, t.fuzzer.Config().Fuzzing.Workers)

	// Create a test case for every sometimes test method.
	for _, contract := range t.fuzzer.ContractDefinitions() {
		// If we're not testing all contracts, verify the current contract is one we specified in our target contracts.
		if !t.fuzzer.config.Fuzzing.Testing.TestAllContracts && !slices.Contains(t.fuzzer.config.Fuzzing.TargetContracts, contract.Name()) {
			continue
		}

		for _, method := range contract.SometimesTestMethods {
			// Create local variables to avoid pointer types in the loop being overridden.
			contract := contract
			method := method

			// Create our sometimes test case
			sometimesTestCase := &SometimesTestCase{
				status:            TestCaseStatusNotStarted,
				targetContract:    contract,
				targetMethod:      method,
				executionCount:    0,
				successCount:      0,
				minSuccessRate:    t.fuzzer.config.Fuzzing.Testing.SometimesTesting.MinSuccessRate,
				minExecutionCount: t.fuzzer.config.Fuzzing.Testing.SometimesTesting.MinExecutionCount,
			}

			// Add to our test cases and register them with the fuzzer
			methodId := contracts.GetContractMethodID(contract, &method)
			t.testCases[methodId] = sometimesTestCase
			t.fuzzer.RegisterTestCase(sometimesTestCase)
		}
	}
	return nil
}

// onFuzzerStopping is the event handler triggered when the Fuzzer is stopping the fuzzing campaign and all workers
// have been destroyed. It clears state tracked for each FuzzerWorker and evaluates test cases to determine
// if they passed or failed based on their success rates.
func (t *SometimesTestCaseProvider) onFuzzerStopping(event FuzzerStoppingEvent) error {
	// Clear our sometimes test methods
	t.workerStates = nil

	// Loop through each test case and evaluate success rates
	for _, testCase := range t.testCases {
		// Only evaluate tests that are running and have enough executions
		if testCase.status == TestCaseStatusRunning {
			if testCase.executionCount >= testCase.minExecutionCount {
				// Calculate success rate
				successRate := testCase.SuccessRate()

				// Check if success rate meets the minimum threshold
				if successRate >= testCase.minSuccessRate {
					testCase.status = TestCaseStatusPassed
				} else {
					testCase.status = TestCaseStatusFailed
					t.fuzzer.ReportTestCaseFinished(testCase)
				}
			} else {
				// Not enough executions to evaluate, mark as passed with a note
				testCase.status = TestCaseStatusPassed
			}
		}
	}
	return nil
}

// onWorkerCreated is the event handler triggered when a FuzzerWorker is created by the Fuzzer. It ensures state tracked
// for that worker index is refreshed and subscribes to relevant worker events.
func (t *SometimesTestCaseProvider) onWorkerCreated(event FuzzerWorkerCreatedEvent) error {
	// Create a new state for this worker.
	t.workerStates[event.Worker.WorkerIndex()] = sometimesTestCaseProviderWorkerState{
		sometimesTestMethods:     make(map[contracts.ContractMethodID]contracts.DeployedContractMethod),
		sometimesTestMethodsLock: sync.Mutex{},
	}

	// Subscribe to relevant worker events.
	event.Worker.Events.ContractAdded.Subscribe(t.onWorkerDeployedContractAdded)
	event.Worker.Events.ContractDeleted.Subscribe(t.onWorkerDeployedContractDeleted)
	return nil
}

// onWorkerDeployedContractAdded is the event handler triggered when a FuzzerWorker detects a new contract deployment
// on its underlying chain. It ensures any sometimes test methods which the deployed contract contains are tracked by the
// provider for testing. Any test cases previously made for these methods which are in a "not started" state are put
// into a "running" state, as they are now potentially reachable for testing.
func (t *SometimesTestCaseProvider) onWorkerDeployedContractAdded(event FuzzerWorkerContractAddedEvent) error {
	// If we don't have a contract definition, we can't run sometimes tests against the contract.
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
		sometimesTestCase, sometimesTestCaseExists := t.testCases[methodId]
		t.testCasesLock.Unlock()

		if sometimesTestCaseExists {
			if sometimesTestCase.Status() == TestCaseStatusNotStarted {
				sometimesTestCase.status = TestCaseStatusRunning
			}
			if sometimesTestCase.Status() != TestCaseStatusFailed {
				// Create our sometimes test method reference.
				workerState := &t.workerStates[event.Worker.WorkerIndex()]
				workerState.sometimesTestMethodsLock.Lock()
				workerState.sometimesTestMethods[methodId] = contracts.DeployedContractMethod{
					Address:  event.ContractAddress,
					Contract: event.ContractDefinition,
					Method:   method,
				}
				workerState.sometimesTestMethodsLock.Unlock()
			}
		}
	}
	return nil
}

// onWorkerDeployedContractDeleted is the event handler triggered when a FuzzerWorker detects that a previously deployed
// contract no longer exists on its underlying chain. It ensures any sometimes test methods which the deployed contract
// contained are no longer tracked by the provider for testing.
func (t *SometimesTestCaseProvider) onWorkerDeployedContractDeleted(event FuzzerWorkerContractDeletedEvent) error {
	// If we don't have a contract definition, there's nothing to do.
	if event.ContractDefinition == nil {
		return nil
	}

	// Loop through all methods and find ones for which we have tests
	for _, method := range event.ContractDefinition.CompiledContract().Abi.Methods {
		// Obtain an identifier for this pair
		methodId := contracts.GetContractMethodID(event.ContractDefinition, &method)

		// If this identifier is in our test cases map, then we remove it from our sometimes test method lookup for
		// this worker index.
		t.testCasesLock.Lock()
		_, isSometimesTestMethod := t.testCases[methodId]
		t.testCasesLock.Unlock()

		if isSometimesTestMethod {
			// Delete our sometimes test method reference.
			workerState := &t.workerStates[event.Worker.WorkerIndex()]
			workerState.sometimesTestMethodsLock.Lock()
			delete(workerState.sometimesTestMethods, methodId)
			workerState.sometimesTestMethodsLock.Unlock()
		}
	}
	return nil
}

// callSequencePostCallTest is a CallSequenceTestFunc that performs post-call testing logic for the attached Fuzzer
// and any underlying FuzzerWorker. It is called after every call made in a call sequence. It executes sometimes
// test methods and tracks their success/failure rates.
func (t *SometimesTestCaseProvider) callSequencePostCallTest(worker *FuzzerWorker, callSequence calls.CallSequence) ([]ShrinkCallSequenceRequest, error) {
	// Obtain the test provider state for this worker
	workerState := &t.workerStates[worker.WorkerIndex()]

	// Loop through all sometimes test methods and test them.
	for sometimesTestMethodId, workerSometimesTestMethod := range workerState.sometimesTestMethods {
		// Obtain the test case for this sometimes test method
		t.testCasesLock.Lock()
		testCase := t.testCases[sometimesTestMethodId]
		t.testCasesLock.Unlock()

		// If the test case already failed, skip it
		if testCase.Status() == TestCaseStatusFailed {
			continue
		}

		// Test our sometimes test method (create a local copy to avoid loop overwriting the method)
		workerSometimesTestMethod := workerSometimesTestMethod
		succeeded, err := t.executeSometimesTest(worker, &workerSometimesTestMethod)
		if err != nil {
			return nil, err
		}

		// Update statistics (thread-safe)
		t.testCasesLock.Lock()
		testCase.executionCount++
		if succeeded {
			testCase.successCount++
		}
		t.testCasesLock.Unlock()
	}

	// Sometimes tests never trigger shrinking, as there's no single failing sequence
	return nil, nil
}
