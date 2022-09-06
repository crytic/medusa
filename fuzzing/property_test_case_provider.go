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
	testCases map[string]*PropertyTestCase

	// propertyTestMethods is a map of workerIndex->contract-method ID -> deployed contract method descriptors.
	// Where each deployed contract method is a property test to evaluate. Property tests should be read-only
	// (pure/view) functions which take  no input parameters and return a boolean variable indicating if the
	// property test passed.
	propertyTestMethods     map[int]map[string]types.DeployedContractMethod
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
func (t *PropertyTestCaseProvider) checkPropertyTestFailed(worker *FuzzerWorker, propertyTestMethod *types.DeployedContractMethod) bool {
	// Generate our ABI input data for the call (just the method ID, no args)
	data, err := propertyTestMethod.Contract.CompiledContract().Abi.Pack(propertyTestMethod.Method.Name)
	if err != nil {
		panic(err)
	}

	// Call the underlying contract
	// TODO: Determine if we should use `Senders[0]` or have a separate funded account for the assertions.
	value := big.NewInt(0)
	msg := worker.TestNode().CreateMessage(worker.Fuzzer().senders[0], &propertyTestMethod.Address, value, data)
	res, err := worker.TestNode().CallContract(msg)
	if err != nil {
		panic(err)
	}

	// If our property test method call failed, we flag a failed test.
	if res.Failed() {
		return true
	}

	// Decode our ABI outputs
	retVals, err := propertyTestMethod.Method.Outputs.Unpack(res.Return())
	if err != nil {
		panic(err)
	}

	// We should have one return value.
	if len(retVals) != 1 {
		panic(fmt.Sprintf("unexpected number of return values in property '%s'", propertyTestMethod.Method.Name))
	}

	// The one return value should be a bool
	propertyTestMethodPassed, ok := retVals[0].(bool)
	if !ok {
		panic(fmt.Sprintf("could not obtain bool from first ABI output element in property '%s'", propertyTestMethod.Method.Name))
	}

	// Return our status from our property test method
	return !propertyTestMethodPassed
}

// OnFuzzerStarting is called by the fuzzing.Fuzzer upon the start of a fuzzing campaign. Any previously recorded
// TestCase should be cleared from the provider and state should be reset.
func (t *PropertyTestCaseProvider) OnFuzzerStarting(fuzzer *Fuzzer) {
	// Reset our state
	t.testCases = make(map[string]*PropertyTestCase)
	t.propertyTestMethods = make(map[int]map[string]types.DeployedContractMethod)

	// Create a test case for every property test method.
	for _, contract := range fuzzer.Contracts() {
		for _, method := range contract.CompiledContract().Abi.Methods {
			if t.isPropertyTest(method, fuzzer.Config().Fuzzing.PropertyTestPrefixes) {
				contract := contract
				method := method
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
}

// OnWorkerCreated is called when a new fuzzing.FuzzerWorker is created by the fuzzing.Fuzzer.
func (t *PropertyTestCaseProvider) OnWorkerCreated(worker *FuzzerWorker) {
	// Lock to avoid concurrent map access issues.
	t.propertyTestMethodsLock.Lock()
	defer t.propertyTestMethodsLock.Unlock()

	// Refresh our property test map for this worker
	t.propertyTestMethods[worker.WorkerIndex()] = make(map[string]types.DeployedContractMethod)
}

// OnWorkerDestroyed is called when a previously created fuzzing.FuzzerWorker is destroyed by the fuzzing.Fuzzer.
func (t *PropertyTestCaseProvider) OnWorkerDestroyed(worker *FuzzerWorker) {}

// OnWorkerDeployedContractAdded is called when a fuzzing.FuzzerWorker detects a newly deployed contract in the
// underlying TestNode. If the  contract could be matched to a definition registered with the fuzzing.Fuzzer,
// it is provided as well. Otherwise, a nil contract definition is supplied.
func (t *PropertyTestCaseProvider) OnWorkerDeployedContractAdded(worker *FuzzerWorker, contractAddress common.Address, contract *types.Contract) {
	// If we don't have a contract definition, we can't run property tests against the contract.
	if contract == nil {
		return
	}

	// Loop through all methods and find ones for which we have tests
	for _, method := range contract.CompiledContract().Abi.Methods {
		// Obtain an identifier for this pair
		methodId := types.GetContractMethodID(contract, &method)

		// If we have a test case targeting this contract/method that has not failed, track this deployed method in
		// our map for this worker.
		if propertyTestCase, exists := t.testCases[methodId]; exists {
			if propertyTestCase.Status() != TestCaseStatusFailed {
				// Lock to avoid concurrent map access issues.
				t.propertyTestMethodsLock.Lock()
				t.propertyTestMethods[worker.WorkerIndex()][methodId] = types.DeployedContractMethod{
					Address:  contractAddress,
					Contract: contract,
					Method:   method,
				}
				t.propertyTestMethodsLock.Unlock()
			}
		}
	}
}

// OnWorkerDeployedContractDeleted is called when a fuzzing.FuzzerWorker detects a previously reported deployed
// contract that no longer exists in the underlying TestNode.
func (t *PropertyTestCaseProvider) OnWorkerDeployedContractDeleted(worker *FuzzerWorker, contractAddress common.Address, contract *types.Contract) {
	// If we don't have a contract definition, there's nothing to do.
	if contract == nil {
		return
	}

	// Loop through all methods and find ones for which we have tests
	for _, method := range contract.CompiledContract().Abi.Methods {
		// Obtain an identifier for this pair
		methodId := types.GetContractMethodID(contract, &method)

		// If this identifier is in our test cases map, then we remove it from our property test method list for
		// this worker index.
		if _, isPropertyTestMethod := t.testCases[methodId]; isPropertyTestMethod {
			delete(t.propertyTestMethods[worker.WorkerIndex()], methodId)
		}
	}
}

// OnWorkerTestedCall is called after a fuzzing.FuzzerWorker sends another call in a types.CallSequence during
// a fuzzing campaign. It returns a ShrinkCallSequenceRequest set, which represents a set of requests for
// shrunken call sequences alongside verifiers to guide the shrinking process.
func (t *PropertyTestCaseProvider) OnWorkerTestedCall(worker *FuzzerWorker, callSequence types.CallSequence) []ShrinkCallSequenceRequest {
	// Lock to avoid concurrent map access issues.
	t.propertyTestMethodsLock.Lock()
	defer t.propertyTestMethodsLock.Unlock()

	// Create a list of shrink call sequence verifiers, which we populate for each failed property test we want a call
	// sequence shrunk for.
	shrinkRequests := make([]ShrinkCallSequenceRequest, 0)

	// Obtain the property test methods for this worker
	workerPropertyTestMethods := t.propertyTestMethods[worker.WorkerIndex()]

	// Loop through all property test methods and test them.
	for workerPropertyTestMethodId, workerPropertyTestMethod := range workerPropertyTestMethods {
		// Obtain the test case for this property test method
		testCase := t.testCases[workerPropertyTestMethodId]

		// If the test case already failed, skip it
		if testCase.Status() == TestCaseStatusFailed {
			continue
		}

		// Test our property test method
		workerPropertyTestMethod := workerPropertyTestMethod
		failedPropertyTest := t.checkPropertyTestFailed(worker, &workerPropertyTestMethod)

		// If we failed a test, we update our state immediately. We provide a shrink verifier which will update
		// the call sequence for each shrunken sequence provided that fails the property test.
		if failedPropertyTest {
			// Create a request to shrink this call sequence.
			shrinkRequest := ShrinkCallSequenceRequest{
				VerifierFunction: func(worker *FuzzerWorker, callSequence types.CallSequence) bool {
					// The shrink verifier simply ensures the previously failed property test fails
					// for the shrunk sequence as well.
					return t.checkPropertyTestFailed(worker, &workerPropertyTestMethod)
				},
				FinishedCallback: func(worker *FuzzerWorker, shrunkenCallSequence types.CallSequence) {
					// When we're finished shrinking, update our test state and report it finalized.
					testCase.status = TestCaseStatusFailed
					testCase.callSequence = shrunkenCallSequence
					worker.Fuzzer().ReportFinishedTestCase(testCase)
				},
			}

			// Add our shrink request to our list.
			shrinkRequests = append(shrinkRequests, shrinkRequest)
		}
	}

	return shrinkRequests
}

// OnFuzzerStopping is called when a fuzzing.Fuzzer's campaign is being stopped. Any TestCase which is still in a running
// state should be updated during this step and put into a finalized state.
func (t *PropertyTestCaseProvider) OnFuzzerStopping() {
	// Clear our property test methods
	t.propertyTestMethodsLock.Lock()
	t.propertyTestMethods = nil
	defer t.propertyTestMethodsLock.Unlock()

	// Loop through each test case and set any tests with a running status to a passed status.
	for _, testCase := range t.testCases {
		if testCase.status == TestCaseStatusRunning {
			testCase.status = TestCaseStatusPassed
		}
	}
}
