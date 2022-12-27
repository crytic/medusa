package fuzzing

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/trailofbits/medusa/fuzzing/types"
	"math/big"
	"strings"
	"sync"
)

// OptimizationTestCaseProvider is a provider for on-chain optimization tests.
// Optimization tests are represented as publicly-accessible view functions which have a name prefix specified by a
// config.FuzzingConfig. They take no input arguments and return an integer value that needs to be maximize.
type OptimizationTestCaseProvider struct {
	// fuzzer describes the Fuzzer which this provider is attached to.
	fuzzer *Fuzzer

	// testCases is a map of contract-method IDs to optimization test cases.GetContractMethodID
	testCases map[types.ContractMethodID]*OptimizationTestCase

	// testCasesLock is used for thread-synchronization when updating testCases
	testCasesLock sync.Mutex

	// workerStates is a slice where each element stores state for a given worker index.
	workerStates []optimizationTestCaseProviderWorkerState
}

// optimizationTestCaseProviderWorkerState represents the state for an individual worker maintained by
// OptimizationTestCaseProvider.
type optimizationTestCaseProviderWorkerState struct {
	// optimizationTestMethods a mapping from contract-method ID to deployed contract-method descriptors.
	// Each deployed contract-method represents a optimization test method to call for evaluation. Optimization tests
	// should be read-only (pure/view) functions which take no input parameters and return an integer variable.
	optimizationTestMethods map[types.ContractMethodID]types.DeployedContractMethod

	// optimizationTestMethodsLock is used for thread-synchronization when updating optimizationTestMethods
	optimizationTestMethodsLock sync.Mutex
}

// attachOptimizationTestCaseProvider attaches a new OptimizationTestCaseProvider to the Fuzzer and returns it.
func attachOptimizationTestCaseProvider(fuzzer *Fuzzer) *OptimizationTestCaseProvider {
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

// isOptimizationTest check whether the method is a optimization test given potential naming prefixes it must conform to
// and its underlying input/output arguments.
func (t *OptimizationTestCaseProvider) isOptimizationTest(method abi.Method) bool {
	// Loop through all enabled prefixes to find a match
	for _, prefix := range t.fuzzer.Config().Fuzzing.Testing.OptimizationTesting.TestPrefixes {
		if strings.HasPrefix(method.Name, prefix) {
			if len(method.Inputs) == 0 && len(method.Outputs) == 1 && method.Outputs[0].Type.T == abi.IntTy {
				return true
			}
		}
	}
	return false
}

// updateOptimizationTest executes a given optimization test method to get the computed value. This is used to
// facilitate updating of optimization test value after every call the Fuzzer makes when testing call sequences.
func (t *OptimizationTestCaseProvider) updateOptimizationTest(worker *FuzzerWorker, optimizationTestMethod *types.DeployedContractMethod, testCase *OptimizationTestCase) error {
	// Generate our ABI input data for the call. In this case, optimization test methods take no arguments, so the
	// variadic argument list here is empty.
	data, err := optimizationTestMethod.Contract.CompiledContract().Abi.Pack(optimizationTestMethod.Method.Name)
	if err != nil {
		return err
	}

	// Call the underlying contract
	// TODO: Determine if we should use `Senders[0]` or have a separate funded account for the optimizations.
	value := big.NewInt(0)
	msg := worker.Chain().CreateMessage(worker.Fuzzer().senders[0], &optimizationTestMethod.Address, value, nil, nil, data)
	res, err := worker.Chain().CallContract(msg)
	if err != nil {
		return fmt.Errorf("failed to call optimization test method: %v", err)
	}

	// If our optimization test method call failed, we flag a failed test.
	// TODO check it during code review, for property tests they mark test failed here
	if res.Failed() {
		return fmt.Errorf("failed to call optimization test method: %v", err)
	}

	// Decode our ABI outputs
	retVals, err := optimizationTestMethod.Method.Outputs.Unpack(res.Return())
	if err != nil {
		return fmt.Errorf("failed to decode optimization test method return value: %v", err)
	}

	// We should have one return value.
	if len(retVals) != 1 {
		return fmt.Errorf("detected an unexpected number of return values from optimization test '%s'", optimizationTestMethod.Method.Name)
	}

	// The one return value should be an integer
	newValue := new(big.Int)
	switch v := retVals[0].(type) {
	case int8:
		newValue.SetInt64(int64(v))
	case int16:
		newValue.SetInt64(int64(v))
	case int32:
		newValue.SetInt64(int64(v))
	case int64:
		newValue.SetInt64(v)
	case uint8:
		newValue.SetUint64(uint64(v))
	case uint16:
		newValue.SetUint64(uint64(v))
	case uint32:
		newValue.SetUint64(uint64(v))
	case uint64:
		newValue.SetUint64(v)
	case *big.Int:
		newValue.Set(v)
	default:
		return fmt.Errorf("failed to parse optimization test method success status from return value '%s'", optimizationTestMethod.Method.Name)
	}

	// Update test case value if the new value is larger than existing value
	if newValue.Cmp(testCase.value) == 1 {
		testCase.value = newValue
	}

	// Return our status from our optimization test method
	return nil
}

// onFuzzerStarting is the event handler triggered when the Fuzzer is starting a fuzzing campaign. It creates test cases
// in a "not started" state for every optimization test method discovered in the contract definitions known to the Fuzzer.
func (t *OptimizationTestCaseProvider) onFuzzerStarting(event FuzzerStartingEvent) error {
	// Reset our state
	t.testCases = make(map[types.ContractMethodID]*OptimizationTestCase)
	t.workerStates = make([]optimizationTestCaseProviderWorkerState, t.fuzzer.Config().Fuzzing.Workers)

	// Create a test case for every optimization test method.
	for _, contract := range t.fuzzer.ContractDefinitions() {
		for _, method := range contract.CompiledContract().Abi.Methods {
			if t.isOptimizationTest(method) {
				// Create local variables to avoid pointer types in the loop being overridden.
				contract := contract
				method := method
				minInt256, _ := new(big.Int).SetString(
					"-8000000000000000000000000000000000000000000000000000000000000000", 16)

				// Create our optimization test case
				optimizationTestCase := &OptimizationTestCase{
					status:         TestCaseStatusNotStarted,
					targetContract: &contract,
					targetMethod:   method,
					callSequence:   nil,
					value:          minInt256,
				}

				// Add to our test cases and register them with the fuzzer
				methodId := types.GetContractMethodID(&contract, &method)
				t.testCases[methodId] = optimizationTestCase
				t.fuzzer.RegisterTestCase(optimizationTestCase)
			}
		}
	}
	return nil
}

// onFuzzerStarting is the event handler triggered when the Fuzzer is stopping the fuzzing campaign and all workers
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
		optimizationTestMethods:     make(map[types.ContractMethodID]types.DeployedContractMethod),
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
		methodId := types.GetContractMethodID(event.ContractDefinition, &method)

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
				workerState.optimizationTestMethods[methodId] = types.DeployedContractMethod{
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

// onWorkerDeployedContractAdded is the event handler triggered when a FuzzerWorker detects that a previously deployed
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
		methodId := types.GetContractMethodID(event.ContractDefinition, &method)

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
// and any underlying FuzzerWorker. It is called after every call made in a call sequence. It checks whether optimization
// test invariants are upheld after each call the Fuzzer makes when testing a call sequence.
func (t *OptimizationTestCaseProvider) callSequencePostCallTest(worker *FuzzerWorker, callSequence types.CallSequence) ([]ShrinkCallSequenceRequest, error) {
	// Obtain the test provider state for this worker
	workerState := &t.workerStates[worker.WorkerIndex()]

	// Loop through all optimization test methods and test them.
	for optimizationTestMethodId, workerOptimizationTestMethod := range workerState.optimizationTestMethods {
		// Obtain the test case for this optimization test method
		t.testCasesLock.Lock()
		testCase := t.testCases[optimizationTestMethodId]
		t.testCasesLock.Unlock()

		// If the test case already failed, skip it
		if testCase.Status() == TestCaseStatusFailed {
			continue
		}

		// Test our optimization test method (create a local copy to avoid loop overwriting the method)
		workerOptimizationTestMethod := workerOptimizationTestMethod
		err := t.updateOptimizationTest(worker, &workerOptimizationTestMethod, testCase)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}
