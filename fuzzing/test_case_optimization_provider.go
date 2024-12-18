package fuzzing

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/crytic/medusa/fuzzing/calls"
	"github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/fuzzing/executiontracer"
	"github.com/ethereum/go-ethereum/core"
	"golang.org/x/exp/slices"
)

const MIN_INT = "-8000000000000000000000000000000000000000000000000000000000000000"

// OptimizationTestCaseProvider is a provider for on-chain optimization tests.
// Optimization tests are represented as publicly-accessible functions which have a name prefix specified by a
// config.FuzzingConfig. They take no input arguments and return an integer value that needs to be maximized.
type OptimizationTestCaseProvider struct {
	// fuzzer describes the Fuzzer which this provider is attached to.
	fuzzer *Fuzzer

	// testCases is a map of contract-method IDs to optimization test cases.GetContractMethodID
	testCases map[contracts.ContractMethodID]*OptimizationTestCase

	// testCasesLock is used for thread-synchronization when updating testCases
	testCasesLock sync.Mutex

	// workerStates is a slice where each element stores state for a given worker index.
	workerStates []optimizationTestCaseProviderWorkerState
}

// optimizationTestCaseProviderWorkerState represents the state for an individual worker maintained by
// OptimizationTestCaseProvider.
type optimizationTestCaseProviderWorkerState struct {
	// optimizationTestMethods a mapping from contract-method ID to deployed contract-method descriptors.
	// Each deployed contract-method represents an optimization test method to call for evaluation. Optimization tests
	// should be read-only functions which take no input parameters and return an integer variable.
	optimizationTestMethods map[contracts.ContractMethodID]contracts.DeployedContractMethod

	// optimizationTestMethodsLock is used for thread-synchronization when updating optimizationTestMethods
	optimizationTestMethodsLock sync.Mutex
}

// attachOptimizationTestCaseProvider attaches a new OptimizationTestCaseProvider to the Fuzzer and returns it.
func attachOptimizationTestCaseProvider(fuzzer *Fuzzer) *OptimizationTestCaseProvider {
	// If there are no testing prefixes, then there is no reason to attach a test case provider and subscribe to events
	if len(fuzzer.config.Fuzzing.Testing.OptimizationTesting.TestPrefixes) == 0 {
		return nil
	}

	// Create a test case provider
	t := &OptimizationTestCaseProvider{
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

// runOptimizationTest executes a given optimization test method (w/ an optional execution trace) and returns the return value
// from the optimization test method. This is called after every call the Fuzzer makes when testing call sequences for each test case.
func (t *OptimizationTestCaseProvider) runOptimizationTest(worker *FuzzerWorker, optimizationTestMethod *contracts.DeployedContractMethod, trace bool) (*big.Int, *executiontracer.ExecutionTrace, error) {
	// Generate our ABI input data for the call. In this case, optimization test methods take no arguments, so the
	// variadic argument list here is empty.
	data, err := optimizationTestMethod.Contract.CompiledContract().Abi.Pack(optimizationTestMethod.Method.Name)
	if err != nil {
		return nil, nil, err
	}

	// Call the underlying contract
	value := big.NewInt(0)
	// TODO: Determine if we should use `Senders[0]` or have a separate funded account for the optimizations.
	msg := calls.NewCallMessage(worker.Fuzzer().senders[0], &optimizationTestMethod.Address, 0, value, worker.fuzzer.config.Fuzzing.TransactionGasLimit, nil, nil, nil, data)
	msg.FillFromTestChainProperties(worker.chain)

	// Execute the call. If we are tracing, we attach an execution tracer and obtain the result.
	var executionResult *core.ExecutionResult
	var executionTrace *executiontracer.ExecutionTrace
	if trace {
		executionResult, executionTrace, err = executiontracer.CallWithExecutionTrace(worker.chain, worker.fuzzer.contractDefinitions, msg.ToCoreMessage(), nil)
	} else {
		executionResult, err = worker.Chain().CallContract(msg.ToCoreMessage(), nil)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to call optimization test method: %v", err)
	}

	// If the execution reverted, then we know that we do not have any valuable return data, so we return the smallest
	// integer value
	if executionResult.Failed() {
		minInt256, _ := new(big.Int).SetString(MIN_INT, 16)
		return minInt256, nil, nil
	}

	// Decode our ABI outputs
	retVals, err := optimizationTestMethod.Method.Outputs.Unpack(executionResult.Return())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode optimization test method return value: %v", err)
	}

	// We should have one return value.
	if len(retVals) != 1 {
		return nil, nil, fmt.Errorf("detected an unexpected number of return values from optimization test '%s'", optimizationTestMethod.Method.Name)
	}

	// Parse the return value and it should be an int256
	newValue, ok := retVals[0].(*big.Int)
	if !ok {
		return nil, nil, fmt.Errorf("failed to parse optimization test's: %s return value: %v", optimizationTestMethod.Method.Name, retVals[0])
	}

	return newValue, executionTrace, nil
}

// onFuzzerStarting is the event handler triggered when the Fuzzer is starting a fuzzing campaign. It creates test cases
// in a "not started" state for every optimization test method discovered in the contract definitions known to the Fuzzer.
func (t *OptimizationTestCaseProvider) onFuzzerStarting(event FuzzerStartingEvent) error {
	// Reset our state
	t.testCases = make(map[contracts.ContractMethodID]*OptimizationTestCase)
	t.workerStates = make([]optimizationTestCaseProviderWorkerState, t.fuzzer.Config().Fuzzing.Workers)

	// Create a test case for every optimization test method.
	for _, contract := range t.fuzzer.ContractDefinitions() {
		// If we're not testing all contracts, verify the current contract is one we specified in our target contracts
		if !t.fuzzer.config.Fuzzing.Testing.TestAllContracts && !slices.Contains(t.fuzzer.config.Fuzzing.TargetContracts, contract.Name()) {
			continue
		}

		for _, method := range contract.OptimizationTestMethods {
			// Create local variables to avoid pointer types in the loop being overridden.
			contract := contract
			method := method
			minInt256, _ := new(big.Int).SetString(MIN_INT, 16)

			// Create our optimization test case
			optimizationTestCase := &OptimizationTestCase{
				status:         TestCaseStatusNotStarted,
				targetContract: contract,
				targetMethod:   method,
				callSequence:   nil,
				value:          minInt256,
			}

			// Add to our test cases and register them with the fuzzer
			methodId := contracts.GetContractMethodID(contract, &method)
			t.testCases[methodId] = optimizationTestCase
			t.fuzzer.RegisterTestCase(optimizationTestCase)
		}
	}
	return nil
}

// onFuzzerStopping is the event handler triggered when the Fuzzer is stopping the fuzzing campaign and all workers
// have been destroyed. It clears state tracked for each FuzzerWorker and sets test cases in "running" states to
// "passed".
func (t *OptimizationTestCaseProvider) onFuzzerStopping(event FuzzerStoppingEvent) error {
	// Clear our optimization test methods
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
func (t *OptimizationTestCaseProvider) onWorkerCreated(event FuzzerWorkerCreatedEvent) error {
	// Create a new state for this worker.
	t.workerStates[event.Worker.WorkerIndex()] = optimizationTestCaseProviderWorkerState{
		optimizationTestMethods:     make(map[contracts.ContractMethodID]contracts.DeployedContractMethod),
		optimizationTestMethodsLock: sync.Mutex{},
	}

	// Subscribe to relevant worker events.
	event.Worker.Events.ContractAdded.Subscribe(t.onWorkerDeployedContractAdded)
	event.Worker.Events.ContractDeleted.Subscribe(t.onWorkerDeployedContractDeleted)
	return nil
}

// onWorkerDeployedContractAdded is the event handler triggered when a FuzzerWorker detects a new contract deployment
// on its underlying chain. It ensures any optimization test methods which the deployed contract contains are tracked by the
// provider for testing. Any test cases previously made for these methods which are in a "not started" state are put
// into a "running" state, as they are now potentially reachable for testing.
func (t *OptimizationTestCaseProvider) onWorkerDeployedContractAdded(event FuzzerWorkerContractAddedEvent) error {
	// If we don't have a contract definition, we can't run optimization tests against the contract.
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
		optimizationTestCase, optimizationTestCaseExists := t.testCases[methodId]
		t.testCasesLock.Unlock()

		if optimizationTestCaseExists {
			if optimizationTestCase.Status() == TestCaseStatusNotStarted {
				optimizationTestCase.status = TestCaseStatusRunning
			}
			if optimizationTestCase.Status() != TestCaseStatusFailed {
				// Create our optimization test method reference.
				workerState := &t.workerStates[event.Worker.WorkerIndex()]
				workerState.optimizationTestMethodsLock.Lock()
				workerState.optimizationTestMethods[methodId] = contracts.DeployedContractMethod{
					Address:  event.ContractAddress,
					Contract: event.ContractDefinition,
					Method:   method,
				}
				workerState.optimizationTestMethodsLock.Unlock()
			}
		}
	}
	return nil
}

// onWorkerDeployedContractDeleted is the event handler triggered when a FuzzerWorker detects that a previously deployed
// contract no longer exists on its underlying chain. It ensures any optimization test methods which the deployed contract
// contained are no longer tracked by the provider for testing.
func (t *OptimizationTestCaseProvider) onWorkerDeployedContractDeleted(event FuzzerWorkerContractDeletedEvent) error {
	// If we don't have a contract definition, there's nothing to do.
	if event.ContractDefinition == nil {
		return nil
	}

	// Loop through all methods and find ones for which we have tests
	for _, method := range event.ContractDefinition.CompiledContract().Abi.Methods {
		// Obtain an identifier for this pair
		methodId := contracts.GetContractMethodID(event.ContractDefinition, &method)

		// If this identifier is in our test cases map, then we remove it from our optimization test method lookup for
		// this worker index.
		t.testCasesLock.Lock()
		_, isOptimizationTestMethod := t.testCases[methodId]
		t.testCasesLock.Unlock()

		if isOptimizationTestMethod {
			// Delete our optimization test method reference.
			workerState := &t.workerStates[event.Worker.WorkerIndex()]
			workerState.optimizationTestMethodsLock.Lock()
			delete(workerState.optimizationTestMethods, methodId)
			workerState.optimizationTestMethodsLock.Unlock()
		}
	}
	return nil
}

// callSequencePostCallTest provides is a CallSequenceTestFunc that performs post-call testing logic for the attached Fuzzer
// and any underlying FuzzerWorker. It is called after every call made in a call sequence. It checks whether any
// optimization test's value has increased.
func (t *OptimizationTestCaseProvider) callSequencePostCallTest(worker *FuzzerWorker, callSequence calls.CallSequence) ([]ShrinkCallSequenceRequest, error) {
	// Create a list of shrink call sequence verifiers, which we populate for each maximized optimization test we want a call
	// sequence shrunk for.
	shrinkRequests := make([]ShrinkCallSequenceRequest, 0)

	// Obtain the test provider state for this worker
	workerState := &t.workerStates[worker.WorkerIndex()]

	// Loop through all optimization test methods and test them.
	for optimizationTestMethodId, workerOptimizationTestMethod := range workerState.optimizationTestMethods {
		// Obtain the test case for this optimization test method
		t.testCasesLock.Lock()
		testCase := t.testCases[optimizationTestMethodId]
		t.testCasesLock.Unlock()

		// Run our optimization test (create a local copy to avoid loop overwriting the method)
		workerOptimizationTestMethod := workerOptimizationTestMethod
		newValue, _, err := t.runOptimizationTest(worker, &workerOptimizationTestMethod, false)
		if err != nil {
			return nil, err
		}

		// If we updated the test case's maximum value, we update our state immediately. We provide a shrink verifier which will update
		// the call sequence for each shrunken sequence provided that still it maintains the maximum value.
		// TODO: This is very inefficient since this runs every time a new max value is found. It would be ideal if we
		//  could perform a one-time shrink request. This code should be refactored when we introduce the high-level
		//  testing API.
		if newValue.Cmp(testCase.value) == 1 {
			// Create a request to shrink this call sequence.
			shrinkRequest := ShrinkCallSequenceRequest{
				VerifierFunction: func(worker *FuzzerWorker, shrunkenCallSequence calls.CallSequence) (bool, error) {
					// First verify the contract to the optimization test is still deployed to call upon.
					_, optimizationTestContractDeployed := worker.deployedContracts[workerOptimizationTestMethod.Address]
					if !optimizationTestContractDeployed {
						// If the contract isn't available, this shrunk sequence likely messed up deployment, so we
						// report it as an invalid solution.
						return false, nil
					}

					// Then the shrink verifier ensures that the maximum value has either stayed the same or, hopefully,
					// increased.
					shrunkenSequenceNewValue, _, err := t.runOptimizationTest(worker, &workerOptimizationTestMethod, false)

					// If the shrunken value is greater than new value, then set new value to the shrunken one so that it
					// can be tracked correctly in the finished callback
					if err == nil && shrunkenSequenceNewValue.Cmp(newValue) == 1 {
						newValue = new(big.Int).Set(shrunkenSequenceNewValue)
					}

					return shrunkenSequenceNewValue.Cmp(newValue) >= 0, err
				},
				FinishedCallback: func(worker *FuzzerWorker, shrunkenCallSequence calls.CallSequence, verboseTracing bool) error {
					// When we're finished shrinking, attach an execution trace to the last call. If verboseTracing is true, attach to all calls.
					if len(shrunkenCallSequence) > 0 {
						_, err = calls.ExecuteCallSequenceWithExecutionTracer(worker.chain, worker.fuzzer.contractDefinitions, shrunkenCallSequence, verboseTracing)
						if err != nil {
							return err
						}
					}

					// Execute the property test a final time, this time obtaining an execution trace
					shrunkenSequenceNewValue, executionTrace, err := t.runOptimizationTest(worker, &workerOptimizationTestMethod, true)
					if err != nil {
						return err
					}

					// If, for some reason, the shrunken sequence lowers the new max value, do not save anything and exit
					if shrunkenSequenceNewValue.Cmp(newValue) < 0 {
						return fmt.Errorf("optimized call sequence failed to maximize value")
					}

					// Update our value with lock
					testCase.valueLock.Lock()
					testCase.value = new(big.Int).Set(shrunkenSequenceNewValue)
					testCase.valueLock.Unlock()

					// Update call sequence and trace
					testCase.callSequence = &shrunkenCallSequence
					testCase.optimizationTestTrace = executionTrace
					return nil
				},
				RecordResultInCorpus: true,
			}

			// Add our shrink request to our list.
			shrinkRequests = append(shrinkRequests, shrinkRequest)
		}
	}

	return shrinkRequests, nil
}
