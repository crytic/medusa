package fuzzing

import (
	"fmt"
	"math/big"
	"slices"

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
	// It is written only in onFuzzerStarting (before workers run) and read-only thereafter.
	testCases map[contracts.ContractMethodID]*SometimesTestCase
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
// have been destroyed. It evaluates test cases to determine if they passed or failed based on their success rates.
func (t *SometimesTestCaseProvider) onFuzzerStopping(event FuzzerStoppingEvent) error {
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

// callSequencePostCallTest is a CallSequenceTestFunc that performs post-call testing logic for the attached Fuzzer
// and any underlying FuzzerWorker. It is called after every call made in a call sequence. It executes sometimes
// test methods and tracks their success/failure rates.
func (t *SometimesTestCaseProvider) callSequencePostCallTest(worker *FuzzerWorker, callSequence calls.CallSequence) ([]ShrinkCallSequenceRequest, error) {
	// Loop through all deployed contracts and test any sometimes test methods.
	for addr, contractDef := range worker.DeployedContracts() {
		for _, method := range contractDef.SometimesTestMethods {
			method := method
			methodId := contracts.GetContractMethodID(contractDef, &method)

			// t.testCases is read-only during fuzzing (written only in onFuzzerStarting).
			testCase, exists := t.testCases[methodId]
			if !exists {
				continue
			}

			testCase.lock.Lock()
			done := testCase.status == TestCaseStatusFailed || testCase.status == TestCaseStatusPassed
			testCase.lock.Unlock()
			if done {
				continue
			}

			addr := addr
			succeeded, err := t.executeSometimesTest(worker, &contracts.DeployedContractMethod{
				Address:  addr,
				Contract: contractDef,
				Method:   method,
			})
			if err != nil {
				return nil, err
			}

			testCase.lock.Lock()
			if testCase.status != TestCaseStatusFailed && testCase.status != TestCaseStatusPassed {
				if testCase.status == TestCaseStatusNotStarted {
					testCase.status = TestCaseStatusRunning
				}
				testCase.executionCount++
				if succeeded {
					testCase.successCount++
				}
			}
			testCase.lock.Unlock()
		}
	}

	// Sometimes tests never trigger shrinking, as there's no single failing sequence
	return nil, nil
}
