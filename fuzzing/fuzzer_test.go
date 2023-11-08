package fuzzing

import (
	"encoding/hex"
	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/events"
	"github.com/crytic/medusa/fuzzing/calls"
	"github.com/crytic/medusa/fuzzing/valuegeneration"
	"github.com/crytic/medusa/utils"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"math/rand"
	"testing"

	"github.com/crytic/medusa/fuzzing/config"
	"github.com/stretchr/testify/assert"
)

// TestFuzzerHooks runs tests to ensure that fuzzer hooks can be modified externally on an API level.
func TestFuzzerHooks(t *testing.T) {
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/assertions/assert_immediate.sol",
		configUpdates: func(config *config.ProjectConfig) {
			config.Fuzzing.DeploymentOrder = []string{"TestContract"}
			config.Fuzzing.Testing.PropertyTesting.Enabled = false
			config.Fuzzing.Testing.AssertionTesting.Enabled = true
		},
		method: func(f *fuzzerTestContext) {
			// Attach to fuzzer hooks which simply set a success state.
			var valueGenOk, chainSetupOk, callSeqTestFuncOk bool
			existingSeqGenConfigFunc := f.fuzzer.Hooks.NewCallSequenceGeneratorConfigFunc
			f.fuzzer.Hooks.NewCallSequenceGeneratorConfigFunc = func(fuzzer *Fuzzer, valueSet *valuegeneration.ValueSet, randomProvider *rand.Rand) (*CallSequenceGeneratorConfig, error) {
				valueGenOk = true
				return existingSeqGenConfigFunc(fuzzer, valueSet, randomProvider)
			}
			existingChainSetupFunc := f.fuzzer.Hooks.ChainSetupFunc
			f.fuzzer.Hooks.ChainSetupFunc = func(fuzzer *Fuzzer, testChain *chain.TestChain) error {
				chainSetupOk = true
				return existingChainSetupFunc(fuzzer, testChain)
			}
			f.fuzzer.Hooks.CallSequenceTestFuncs = append(f.fuzzer.Hooks.CallSequenceTestFuncs, func(worker *FuzzerWorker, callSequence calls.CallSequence) ([]ShrinkCallSequenceRequest, error) {
				callSeqTestFuncOk = true
				return make([]ShrinkCallSequenceRequest, 0), nil
			})

			// Start the fuzzer
			err := f.fuzzer.Start()
			assert.NoError(t, err)

			// Check for failed assertion tests.
			assertFailedTestsExpected(f, true)

			// Assert that our hooks worked
			assert.True(t, valueGenOk, "could not hook value generator func")
			assert.True(t, chainSetupOk, "could not hook chain setup func")
			assert.True(t, callSeqTestFuncOk, "could not hook call sequence test func")
		},
	})
}

// TestAssertionMode runs tests to ensure that assertion testing behaves as expected.
func TestAssertionMode(t *testing.T) {
	filePaths := []string{
		"testdata/contracts/assertions/assert_immediate.sol",
		"testdata/contracts/assertions/assert_even_number.sol",
		"testdata/contracts/assertions/assert_arithmetic_underflow.sol",
		"testdata/contracts/assertions/assert_divide_by_zero.sol",
		"testdata/contracts/assertions/assert_enum_type_conversion_outofbounds.sol",
		"testdata/contracts/assertions/assert_incorrect_storage_access.sol",
		"testdata/contracts/assertions/assert_pop_empty_array.sol",
		"testdata/contracts/assertions/assert_outofbounds_array_access.sol",
		"testdata/contracts/assertions/assert_allocate_too_much_memory.sol",
		"testdata/contracts/assertions/assert_call_uninitialized_variable.sol",
	}
	for _, filePath := range filePaths {
		runFuzzerTest(t, &fuzzerSolcFileTest{
			filePath: filePath,
			configUpdates: func(config *config.ProjectConfig) {
				config.Fuzzing.DeploymentOrder = []string{"TestContract"}
				config.Fuzzing.Testing.PropertyTesting.Enabled = false
				config.Fuzzing.Testing.AssertionTesting.Enabled = true
				config.Fuzzing.Testing.AssertionTesting.AssertionModes.FailOnAssertion = true
				config.Fuzzing.Testing.AssertionTesting.AssertionModes.FailOnAllocateTooMuchMemory = true
				config.Fuzzing.Testing.AssertionTesting.AssertionModes.FailOnArithmeticUnderflow = true
				config.Fuzzing.Testing.AssertionTesting.AssertionModes.FailOnCallUninitializedVariable = true
				config.Fuzzing.Testing.AssertionTesting.AssertionModes.FailOnEnumTypeConversionOutOfBounds = true
				config.Fuzzing.Testing.AssertionTesting.AssertionModes.FailOnDivideByZero = true
				config.Fuzzing.Testing.AssertionTesting.AssertionModes.FailOnIncorrectStorageAccess = true
				config.Fuzzing.Testing.AssertionTesting.AssertionModes.FailOnOutOfBoundsArrayAccess = true
				config.Fuzzing.Testing.AssertionTesting.AssertionModes.FailOnPopEmptyArray = true
			},
			method: func(f *fuzzerTestContext) {
				// Start the fuzzer
				err := f.fuzzer.Start()
				assert.NoError(t, err)
				// Check for failed assertion tests.
				assertFailedTestsExpected(f, true)
			},
		})
	}
}

// TestAssertionsNotRequire runs a test to ensure require and revert statements are not mistaken for assert statements.
// It runs tests against a contract which immediately makes these statements and expects to find no errors before
// timing out.
func TestAssertionsNotRequire(t *testing.T) {
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/assertions/assert_not_require.sol",
		configUpdates: func(config *config.ProjectConfig) {
			config.Fuzzing.DeploymentOrder = []string{"TestContract"}
			config.Fuzzing.TestLimit = 500
			config.Fuzzing.Testing.PropertyTesting.Enabled = false
			config.Fuzzing.Testing.AssertionTesting.Enabled = true
		},
		method: func(f *fuzzerTestContext) {
			// Start the fuzzer
			err := f.fuzzer.Start()
			assert.NoError(t, err)

			// Check for failed assertion tests. We expect none.
			assertFailedTestsExpected(f, false)
		},
	})
}

// TestAssertionsAndProperties runs a test to property testing and assertion testing can both run in parallel.
// This test does not stop on first failure and expects a failure from each after timeout.
func TestAssertionsAndProperties(t *testing.T) {
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/assertions/assert_and_property_test.sol",
		configUpdates: func(config *config.ProjectConfig) {
			config.Fuzzing.DeploymentOrder = []string{"TestContract"}
			config.Fuzzing.TestLimit = 500
			config.Fuzzing.Testing.StopOnFailedTest = false
			config.Fuzzing.Testing.PropertyTesting.Enabled = true
			config.Fuzzing.Testing.AssertionTesting.Enabled = true
		},
		method: func(f *fuzzerTestContext) {
			// Start the fuzzer
			err := f.fuzzer.Start()
			assert.NoError(t, err)

			// Check for failed assertion tests. We expect none.
			assert.EqualValues(f.t, 2, len(f.fuzzer.TestCasesWithStatus(TestCaseStatusFailed)), "Expected one failure from a property test, and one failure from an assertion test.")
		},
	})
}

// TestOptimizationMode runs a test to ensure that optimization mode works as expected
func TestOptimizationMode(t *testing.T) {
	filePaths := []string{
		"testdata/contracts/optimizations/optimize.sol",
	}
	for _, filePath := range filePaths {
		runFuzzerTest(t, &fuzzerSolcFileTest{
			filePath: filePath,
			configUpdates: func(config *config.ProjectConfig) {
				config.Fuzzing.DeploymentOrder = []string{"TestContract"}
				config.Fuzzing.Testing.PropertyTesting.Enabled = false
				config.Fuzzing.Testing.AssertionTesting.Enabled = false
				config.Fuzzing.Testing.OptimizationTesting.Enabled = true
				config.Fuzzing.TestLimit = 10_000 // this test should expose a failure quickly.
			},
			method: func(f *fuzzerTestContext) {
				// Start the fuzzer
				err := f.fuzzer.Start()
				assert.NoError(t, err)

				// Check the value found for optimization test
				var testCases = f.fuzzer.TestCasesWithStatus(TestCaseStatusPassed)
				switch v := testCases[0].(type) {
				case *OptimizationTestCase:
					assert.EqualValues(t, v.Value().Cmp(big.NewInt(4241)), 0)
				default:
					t.Errorf("invalid test case found %T", v)
				}
			},
		})
	}
}

// TestChainBehaviour runs tests to ensure the chain behaves as expected.
func TestChainBehaviour(t *testing.T) {
	// Run a test to simulate out of gas errors to make sure its handled well by the Chain and does not panic.
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/chain/tx_out_of_gas.sol",
		configUpdates: func(config *config.ProjectConfig) {
			config.Fuzzing.DeploymentOrder = []string{"TestContract"}
			config.Fuzzing.Workers = 1
			config.Fuzzing.TestLimit = uint64(config.Fuzzing.CallSequenceLength) // we just need a few oog txs to test
			config.Fuzzing.Timeout = 10                                          // to be safe, we set a 10s timeout
			config.Fuzzing.TransactionGasLimit = 500000                          // we set this low, so contract execution runs out of gas earlier.
		},
		method: func(f *fuzzerTestContext) {
			// Start the fuzzer
			err := f.fuzzer.Start()
			assert.NoError(t, err)

			// Assert that we should not have failures.
			assertFailedTestsExpected(f, false)
		},
	})
}

// TestCheatCodes runs tests to ensure that vm extensions ("cheat codes") are working as intended.
func TestCheatCodes(t *testing.T) {
	filePaths := []string{
		"testdata/contracts/cheat_codes/utils/addr.sol",
		"testdata/contracts/cheat_codes/utils/to_string.sol",
		"testdata/contracts/cheat_codes/utils/sign.sol",
		"testdata/contracts/cheat_codes/utils/parse.sol",
		"testdata/contracts/cheat_codes/vm/coinbase.sol",
		"testdata/contracts/cheat_codes/vm/chain_id.sol",
		"testdata/contracts/cheat_codes/vm/deal.sol",
		"testdata/contracts/cheat_codes/vm/difficulty.sol",
		"testdata/contracts/cheat_codes/vm/etch.sol",
		"testdata/contracts/cheat_codes/vm/fee.sol",
		"testdata/contracts/cheat_codes/vm/prank.sol",
		"testdata/contracts/cheat_codes/vm/roll.sol",
		"testdata/contracts/cheat_codes/vm/store_load.sol",
		"testdata/contracts/cheat_codes/vm/warp.sol",
	}

	// FFI test will fail on Windows because "echo" is a shell command, not a system command, so we diverge these
	// tests.
	if utils.IsWindowsEnvironment() {
		filePaths = append(filePaths,
			"testdata/contracts/cheat_codes/utils/ffi_windows.sol",
		)
	} else {
		filePaths = append(filePaths,
			"testdata/contracts/cheat_codes/utils/ffi_unix.sol",
		)
	}

	for _, filePath := range filePaths {
		runFuzzerTest(t, &fuzzerSolcFileTest{
			filePath: filePath,
			configUpdates: func(config *config.ProjectConfig) {
				config.Fuzzing.DeploymentOrder = []string{"TestContract"}

				// some tests require full sequence + revert to test fully
				config.Fuzzing.Workers = 3
				config.Fuzzing.TestLimit = uint64(config.Fuzzing.CallSequenceLength*config.Fuzzing.Workers) * 3

				// enable assertion testing only
				config.Fuzzing.Testing.PropertyTesting.Enabled = false
				config.Fuzzing.Testing.AssertionTesting.Enabled = true
				config.Fuzzing.TestChainConfig.CheatCodeConfig.EnableFFI = true
			},
			method: func(f *fuzzerTestContext) {
				// Start the fuzzer
				err := f.fuzzer.Start()
				assert.NoError(t, err)

				// Check for failed assertion tests.
				assertFailedTestsExpected(f, false)
			},
		})
	}
}

// TestConsoleLog tests the console.log precompile contract by logging a variety of different primitive types and
// then failing. The execution trace for the failing call sequence should hold the various logs.
func TestConsoleLog(t *testing.T) {
	// These are the logs that should show up in the execution trace
	expectedLogs := []string{
		"2",
		"hello world",
		"byte",
		"i is 2",
		"% bool is true, addr is 0x0000000000000000000000000000000000000000, u is 100",
	}

	filePaths := []string{
		"testdata/contracts/cheat_codes/console_log/console_log.sol",
	}
	for _, filePath := range filePaths {
		runFuzzerTest(t, &fuzzerSolcFileTest{
			filePath: filePath,
			configUpdates: func(config *config.ProjectConfig) {
				config.Fuzzing.DeploymentOrder = []string{"TestContract"}
				config.Fuzzing.TestLimit = 10000
				// enable assertion testing only
				config.Fuzzing.Testing.PropertyTesting.Enabled = true
				config.Fuzzing.Testing.AssertionTesting.Enabled = true
			},
			method: func(f *fuzzerTestContext) {
				// Start the fuzzer
				err := f.fuzzer.Start()
				assert.NoError(t, err)

				// Check for failed assertion tests.
				failedTestCase := f.fuzzer.TestCasesWithStatus(TestCaseStatusFailed)
				assert.NotEmpty(t, failedTestCase, "expected to have failed test cases")

				// Obtain our first failed test case, get the message, and verify it contains our assertion failed.
				failingSequence := *failedTestCase[0].CallSequence()
				assert.NotEmpty(t, failingSequence, "expected to have calls in the call sequence failing an assertion test")

				// Obtain the last call
				lastCall := failingSequence[len(failingSequence)-1]
				assert.NotNilf(t, lastCall.ExecutionTrace, "expected to have an execution trace attached to call sequence for this test")

				// Get the execution trace message
				executionTraceMsg := lastCall.ExecutionTrace.Log().String()

				// Verify it contains all expected logs
				for _, expectedLog := range expectedLogs {
					assert.Contains(t, executionTraceMsg, expectedLog)
				}
			},
		})
	}
}

// TestDeploymentsInnerDeployments runs tests to ensure dynamically deployed contracts are detected by the Fuzzer and
// their properties are tested appropriately.
func TestDeploymentsInnerDeployments(t *testing.T) {
	// These contracts provide functions to deploy inner contracts which have properties that will produce a failure.
	filePaths := []string{
		"testdata/contracts/deployments/inner_deployment.sol",
		"testdata/contracts/deployments/inner_inner_deployment.sol",
	}
	for _, filePath := range filePaths {
		runFuzzerTest(t, &fuzzerSolcFileTest{
			filePath: filePath,
			configUpdates: func(config *config.ProjectConfig) {
				config.Fuzzing.DeploymentOrder = []string{"InnerDeploymentFactory"}
				config.Fuzzing.TestLimit = 1_000 // this test should expose a failure quickly.
				config.Fuzzing.Testing.StopOnFailedContractMatching = true
				config.Fuzzing.Testing.TestAllContracts = true // test dynamically deployed contracts
			},
			method: func(f *fuzzerTestContext) {
				// Start the fuzzer
				err := f.fuzzer.Start()
				assert.NoError(t, err)

				// Check for any failed tests and verify coverage was captured
				assertFailedTestsExpected(f, true)
				assertCorpusCallSequencesCollected(f, true)
			},
		})
	}

	// This contract deploys an inner contract upon construction, which contains properties that will produce a failure.
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/deployments/inner_deployment_on_construction.sol",
		configUpdates: func(config *config.ProjectConfig) {
			config.Fuzzing.DeploymentOrder = []string{"InnerDeploymentFactory"}
			config.Fuzzing.TestLimit = 1_000 // this test should expose a failure quickly.
			config.Fuzzing.Testing.StopOnFailedContractMatching = true
			config.Fuzzing.Testing.TestAllContracts = true // test dynamically deployed contracts
		},
		method: func(f *fuzzerTestContext) {
			// Start the fuzzer
			err := f.fuzzer.Start()
			assert.NoError(t, err)

			// Check to see if there are any failures
			assertFailedTestsExpected(f, true)
		},
	})
}

// TestDeploymentsInternalLibrary runs a test to ensure internal libraries behave correctly.
func TestDeploymentsInternalLibrary(t *testing.T) {
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/deployments/internal_library.sol",
		configUpdates: func(config *config.ProjectConfig) {
			config.Fuzzing.DeploymentOrder = []string{"TestInternalLibrary"}
			config.Fuzzing.TestLimit = 100 // this test should expose a failure quickly.
		},
		method: func(f *fuzzerTestContext) {
			// Start the fuzzer
			err := f.fuzzer.Start()
			assert.NoError(t, err)

			// Check for any failed tests and verify coverage was captured
			assertFailedTestsExpected(f, false)
			assertCorpusCallSequencesCollected(f, true)
		},
	})
}

// TestDeploymentsInnerDeployments runs a test to ensure dynamically deployed contracts are detected by the Fuzzer and
// their properties are tested appropriately.
func TestDeploymentsSelfDestruct(t *testing.T) {
	// These contracts provide functions to deploy inner contracts which have properties that will produce a failure.
	filePaths := []string{
		"testdata/contracts/deployments/selfdestruct_init.sol",
		"testdata/contracts/deployments/selfdestruct_runtime.sol",
	}
	for _, filePath := range filePaths {
		runFuzzerTest(t, &fuzzerSolcFileTest{
			filePath: filePath,
			configUpdates: func(config *config.ProjectConfig) {
				config.Fuzzing.DeploymentOrder = []string{"InnerDeploymentFactory"}
				config.Fuzzing.TestLimit = 500 // this test should expose a failure quickly.
				config.Fuzzing.Testing.StopOnNoTests = false
				config.Fuzzing.Testing.TestAllContracts = true
			},
			method: func(f *fuzzerTestContext) {
				// Subscribe to any mined block events globally. When receiving them, check contract changes for a
				// self-destruct.
				selfDestructCount := 0
				events.SubscribeAny(func(event chain.PendingBlockCommittedEvent) error {
					for _, messageResults := range event.Block.MessageResults {
						for _, contractDeploymentChange := range messageResults.ContractDeploymentChanges {
							if contractDeploymentChange.SelfDestructed {
								selfDestructCount++
							}
						}
					}
					return nil
				})

				// Start the fuzzer
				err := f.fuzzer.Start()
				assert.NoError(t, err)

				// When it's done, we should've had at least one self-destruction.
				assert.Greater(t, selfDestructCount, 0, "no SELFDESTRUCT operations were detected, when they should have been.")
			},
		})
	}
}

// TestExecutionTraces runs tests to ensure that execution traces capture information
// regarding assertion failures, revert reasons, etc.
func TestExecutionTraces(t *testing.T) {
	expectedMessagesPerTest := map[string][]string{
		"testdata/contracts/execution_tracing/call_and_deployment_args.sol": {"Hello from deployment args!", "Hello from call args!"},
		"testdata/contracts/execution_tracing/cheatcodes.sol":               {"StdCheats.toString(true)"},
		"testdata/contracts/execution_tracing/event_emission.sol":           {"TestEvent", "TestIndexedEvent", "TestMixedEvent", "Hello from event args!", "Hello from library event args!"},
		"testdata/contracts/execution_tracing/proxy_call.sol":               {"TestContract -> InnerDeploymentContract.setXY", "Hello from proxy call args!"},
		"testdata/contracts/execution_tracing/revert_custom_error.sol":      {"CustomError", "Hello from a custom error!"},
		"testdata/contracts/execution_tracing/revert_reasons.sol":           {"RevertingContract was called and reverted."},
		"testdata/contracts/execution_tracing/self_destruct.sol":            {"[selfdestruct]", "[panic: assertion failed]"},
	}
	for filePath, expectedTraceMessages := range expectedMessagesPerTest {
		runFuzzerTest(t, &fuzzerSolcFileTest{
			filePath: filePath,
			configUpdates: func(config *config.ProjectConfig) {
				config.Fuzzing.DeploymentOrder = []string{"TestContract"}
				config.Fuzzing.Testing.PropertyTesting.Enabled = false
				config.Fuzzing.Testing.AssertionTesting.Enabled = true
			},
			method: func(f *fuzzerTestContext) {
				// Start the fuzzer
				err := f.fuzzer.Start()
				assert.NoError(t, err)

				// Check for failed assertion tests.
				failedTestCase := f.fuzzer.TestCasesWithStatus(TestCaseStatusFailed)
				assert.NotEmpty(t, failedTestCase, "expected to have failed test cases")

				// Obtain our first failed test case, get the message, and verify it contains our assertion failed.
				failingSequence := *failedTestCase[0].CallSequence()
				assert.NotEmpty(t, failingSequence, "expected to have calls in the call sequence failing an assertion test")

				// Obtain the last call
				lastCall := failingSequence[len(failingSequence)-1]
				assert.NotNilf(t, lastCall.ExecutionTrace, "expected to have an execution trace attached to call sequence for this test")

				// Get the execution trace message
				executionTraceMsg := lastCall.ExecutionTrace.Log().String()

				// Verify it contains all expected strings
				for _, expectedTraceMessage := range expectedTraceMessages {
					assert.Contains(t, executionTraceMsg, expectedTraceMessage)
				}
			},
		})
	}
}

// TestTestingScope runs tests to ensure dynamically deployed contracts are tested when the "test all contracts"
// config option is specified. It also runs the fuzzer without the option enabled to ensure they are not tested.
func TestTestingScope(t *testing.T) {
	for _, testingAllContracts := range []bool{false, true} {
		runFuzzerTest(t, &fuzzerSolcFileTest{
			filePath: "testdata/contracts/deployments/testing_scope.sol",
			configUpdates: func(config *config.ProjectConfig) {
				config.Fuzzing.DeploymentOrder = []string{"TestContract"}
				config.Fuzzing.TestLimit = 1_000 // this test should expose a failure quickly.
				config.Fuzzing.Testing.TestAllContracts = testingAllContracts
				config.Fuzzing.Testing.StopOnFailedTest = false
				config.Fuzzing.Testing.AssertionTesting.Enabled = true
				config.Fuzzing.Testing.PropertyTesting.Enabled = true
			},
			method: func(f *fuzzerTestContext) {
				// Start the fuzzer
				err := f.fuzzer.Start()
				assert.NoError(t, err)

				// Define our expected failure count
				var expectedFailureCount int
				if testingAllContracts {
					expectedFailureCount = 4
				} else {
					expectedFailureCount = 2
				}

				// Check for any failed tests and verify coverage was captured
				assert.EqualValues(t, len(f.fuzzer.TestCasesWithStatus(TestCaseStatusFailed)), expectedFailureCount)
			},
		})
	}
}

// TestDeploymentsWithArgs runs tests to ensure contracts deployed with config provided constructor arguments are
// deployed as expected. It expects all properties should fail (indicating values provided were set accordingly).
func TestDeploymentsWithArgs(t *testing.T) {
	// This contract deploys a contract with specific constructor arguments. Property tests will fail if they are
	// set correctly.
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/deployments/deployment_with_args.sol",
		configUpdates: func(config *config.ProjectConfig) {
			config.Fuzzing.DeploymentOrder = []string{"DeploymentWithArgs", "Dependent"}
			config.Fuzzing.ConstructorArgs = map[string]map[string]any{
				"DeploymentWithArgs": {
					"_x": "123456789",
					"_y": "0x5465",
					"_z": map[string]any{
						"a": "0x4d2",
						"b": "0x54657374206465706c6f796d656e74207769746820617267756d656e7473",
					},
				},
				"Dependent": {
					"_deployed": "DeployedContract:DeploymentWithArgs",
				},
			}
			config.Fuzzing.Testing.StopOnFailedTest = false
			config.Fuzzing.TestLimit = 500 // this test should expose a failure quickly.
		},
		method: func(f *fuzzerTestContext) {
			// Start the fuzzer
			err := f.fuzzer.Start()
			assert.NoError(t, err)

			// Check to see if there are any failures
			assert.EqualValues(t, len(f.fuzzer.TestCasesWithStatus(TestCaseStatusFailed)), 4)
		},
	})
}

// TestValueGenerationGenerateAllTypes runs a test to ensure various types of fuzzer inputs can be generated.
func TestValueGenerationGenerateAllTypes(t *testing.T) {
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/value_generation/generate_all_types.sol",
		configUpdates: func(config *config.ProjectConfig) {
			config.Fuzzing.DeploymentOrder = []string{"GenerateAllTypes"}
			config.Fuzzing.TestLimit = 10_000
		},
		method: func(f *fuzzerTestContext) {
			// Start the fuzzer
			err := f.fuzzer.Start()
			assert.NoError(t, err)

			// Check for any failed tests and verify coverage was captured
			assertFailedTestsExpected(f, false)
			assertCorpusCallSequencesCollected(f, true)
		},
	})
}

// TestValueGenerationSolving runs a series of tests to test the value generator can solve expected problems.
func TestValueGenerationSolving(t *testing.T) {
	// TODO: match_ints_xy is slower than match_uints_xy in the value generator because AST doesn't retain negative
	//  numbers, improve our logic to solve it faster, then re-enable this.
	filePaths := []string{
		"testdata/contracts/value_generation/match_addr_contract.sol",
		"testdata/contracts/value_generation/match_addr_exact.sol",
		"testdata/contracts/value_generation/match_addr_sender.sol",
		"testdata/contracts/value_generation/match_string_exact.sol",
		"testdata/contracts/value_generation/match_structs_xy.sol",
		//"testdata/contracts/value_generation/match_ints_xy.sol",
		"testdata/contracts/value_generation/match_uints_xy.sol",
		"testdata/contracts/value_generation/match_payable_xy.sol",
	}
	for _, filePath := range filePaths {
		runFuzzerTest(t, &fuzzerSolcFileTest{
			filePath: filePath,
			configUpdates: func(config *config.ProjectConfig) {
				config.Fuzzing.DeploymentOrder = []string{"TestContract"}
			},
			method: func(f *fuzzerTestContext) {
				// Start the fuzzer
				err := f.fuzzer.Start()
				assert.NoError(t, err)

				// Check for any failed tests and verify coverage was captured
				assertFailedTestsExpected(f, true)
				assertCorpusCallSequencesCollected(f, true)
			},
		})
	}
}

// TestASTValueExtraction runs a test to ensure appropriate AST values can be mined out of a compiled source's AST.
func TestASTValueExtraction(t *testing.T) {
	// Define our expected values to be mined.
	expectedAddresses := []common.Address{
		common.HexToAddress("0x7109709ECfa91a80626fF3989D68f67F5b1DD12D"),
		common.HexToAddress("0x1234567890123456789012345678901234567890"),
	}
	expectedIntegers := []string{
		// Unsigned integer tests
		"111",                 // no denomination
		"1",                   // 1 wei (base unit)
		"2000000000",          // 2 gwei
		"5000000000000000000", // 5 ether
		"6",                   // 6 seconds (base unit)
		"420",                 // 7 minutes
		"28800",               // 8 hours
		"777600",              // 9 days
		"6048000",             // 10 weeks

		// Signed integer tests
		"-111",                 // no denomination
		"-1",                   // 1 wei (base unit)
		"-2000000000",          // 2 gwei
		"-5000000000000000000", // 5 ether
		"-6",                   // 6 seconds (base unit)
		"-420",                 // 7 minutes
		"-28800",               // 8 hours
		"-777600",              // 9 days
		"-6048000",             // 10 weeks
	}
	expectedStrings := []string{
		"testString",
		"testString2",
	}
	expectedByteSequences := make([][]byte, 0) // no tests yet

	// Run the fuzzer test
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/value_generation/ast_value_extraction.sol",
		configUpdates: func(config *config.ProjectConfig) {
			config.Fuzzing.TestLimit = 1 // stop immediately to simply see what values were mined.
			config.Fuzzing.Testing.AssertionTesting.Enabled = true
			config.Fuzzing.Testing.PropertyTesting.Enabled = false
		},
		method: func(f *fuzzerTestContext) {
			// Start the fuzzer
			err := f.fuzzer.Start()
			assert.NoError(t, err)

			// Verify all of our expected values exist
			valueSet := f.fuzzer.BaseValueSet()
			for _, expectedAddr := range expectedAddresses {
				assert.True(t, valueSet.ContainsAddress(expectedAddr), "Value set did not contain expected address: %v", expectedAddr.String())
			}
			for _, expectedIntegerStr := range expectedIntegers {
				expectedInteger, ok := new(big.Int).SetString(expectedIntegerStr, 10)
				assert.True(t, ok, "Could not parse provided expected integer string in test: \"%v\"", expectedIntegerStr)
				assert.True(t, valueSet.ContainsInteger(expectedInteger), "Value set did not contain expected integer: %v", expectedInteger.String())
			}
			for _, expectedString := range expectedStrings {
				assert.True(t, valueSet.ContainsString(expectedString), "Value set did not contain expected string: \"%v\"", expectedString)
			}
			for _, expectedByteSequence := range expectedByteSequences {
				assert.True(t, valueSet.ContainsBytes(expectedByteSequence), "Value set did not contain expected bytes: \"%v\"", hex.EncodeToString(expectedByteSequence))
			}
		},
	})
}

// TestVMCorrectness runs tests to ensure block properties are reported consistently within the EVM, as it's configured
// by the chain.TestChain.
func TestVMCorrectness(t *testing.T) {
	// Test block numbers work as expected.
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/vm_tests/block_number_increasing.sol",
		configUpdates: func(config *config.ProjectConfig) {
			config.Fuzzing.DeploymentOrder = []string{"TestContract"}
			config.Fuzzing.MaxBlockTimestampDelay = 1 // this contract require calls every block
			config.Fuzzing.MaxBlockNumberDelay = 1    // this contract require calls every block
		},
		method: func(f *fuzzerTestContext) {
			// Start the fuzzer
			err := f.fuzzer.Start()
			assert.NoError(t, err)

			// Check for any failed tests and verify coverage was captured
			assertFailedTestsExpected(f, true)
			assertCorpusCallSequencesCollected(f, true)
		},
	})

	// Test timestamps behave as expected.
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/vm_tests/block_number_increasing.sol",
		configUpdates: func(config *config.ProjectConfig) {
			config.Fuzzing.DeploymentOrder = []string{"TestContract"}
			config.Fuzzing.MaxBlockTimestampDelay = 1 // this contract require calls every block
			config.Fuzzing.MaxBlockNumberDelay = 1    // this contract require calls every block
		},
		method: func(f *fuzzerTestContext) {
			// Start the fuzzer
			err := f.fuzzer.Start()
			assert.NoError(t, err)

			// Check for any failed tests and verify coverage was captured
			assertFailedTestsExpected(f, true)
			assertCorpusCallSequencesCollected(f, true)
		},
	})

	// Test block hashes are reported consistently.
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/vm_tests/block_hash_store_check.sol",
		configUpdates: func(config *config.ProjectConfig) {
			config.Fuzzing.DeploymentOrder = []string{"TestContract"}
			config.Fuzzing.TestLimit = 1_000          // this test should expose a failure quickly.
			config.Fuzzing.MaxBlockTimestampDelay = 1 // this contract require calls every block
			config.Fuzzing.MaxBlockNumberDelay = 1    // this contract require calls every block
		},
		method: func(f *fuzzerTestContext) {
			// Start the fuzzer
			err := f.fuzzer.Start()
			assert.NoError(t, err)

			// Check for any failed tests and verify coverage was captured
			assertFailedTestsExpected(f, false)
			assertCorpusCallSequencesCollected(f, true)
		},
	})
}

// TestCorpusReplayability will test whether the corpus, when replayed, will end up with the same coverage.
// Additionally, check if the second run is solved with sequences executed being less or equal to the total corpus
// call sequences. This should occur as the corpus call sequences should be executed unmodified first (including
// the sequence which previously failed the on-chain test), prior to generating any new fuzzed sequences.
func TestCorpusReplayability(t *testing.T) {
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/value_generation/match_uints_xy.sol",
		configUpdates: func(config *config.ProjectConfig) {
			config.Fuzzing.DeploymentOrder = []string{"TestContract"}
			config.Fuzzing.CorpusDirectory = "corpus"
		},
		method: func(f *fuzzerTestContext) {
			// Setup checks for event emissions
			expectEventEmitted(f, &f.fuzzer.Events.FuzzerStarting)

			// Start the fuzzer
			err := f.fuzzer.Start()
			assert.NoError(t, err)

			// Make sure we have some coverage
			assertCorpusCallSequencesCollected(f, true)

			// Cache current coverage maps
			originalCoverage := f.fuzzer.corpus.CoverageMaps()
			originalCorpusSequenceCount := f.fuzzer.corpus.CallSequenceEntryCount(true, true, true)

			// Next, set the fuzzer worker count to one, this allows us to count the call sequences executed before
			// solving a problem. We will verify the problem is solved with less or equal sequences tested, than
			// corpus call sequence items (as the failing test corpus items should be replayed by the call sequence
			// generator prior to it generating any new sequences).
			f.fuzzer.config.Fuzzing.Workers = 1
			err = f.fuzzer.Start()
			assert.NoError(t, err)

			// Check to see if we have some coverage
			assertCorpusCallSequencesCollected(f, true)
			newCoverage := f.fuzzer.corpus.CoverageMaps()

			// Check to see if original and new coverage are the same.
			assert.True(t, originalCoverage.Equal(newCoverage))

			// Verify that the fuzzer finished after fewer sequences than there are in the corpus
			assert.LessOrEqual(t, f.fuzzer.metrics.SequencesTested().Uint64(), uint64(originalCorpusSequenceCount))
		},
	})
}

// TestDeploymentOrderWithCoverage will ensure that changing the deployment order does not lead to the same coverage
// This is also proof that changing the order changes the addresses of the contracts leading to the coverage not being
// useful.
func TestDeploymentOrderWithCoverage(t *testing.T) {
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/deployments/deployment_order.sol",
		configUpdates: func(config *config.ProjectConfig) {
			config.Fuzzing.DeploymentOrder = []string{"InheritedFirstContract", "InheritedSecondContract"}
		},
		method: func(f *fuzzerTestContext) {
			// Setup checks for event emissions
			expectEventEmitted(f, &f.fuzzer.Events.FuzzerStarting)

			// Start the fuzzer
			err := f.fuzzer.Start()
			assert.NoError(t, err)

			// Make sure we have some coverage
			assertCorpusCallSequencesCollected(f, true)

			// Cache current coverage maps
			originalCoverage := f.fuzzer.corpus.CoverageMaps()

			// Subscribe to the event and stop the fuzzer
			f.fuzzer.Events.FuzzerStarting.Subscribe(func(event FuzzerStartingEvent) error {
				// Simply stop the fuzzer
				event.Fuzzer.Stop()
				return nil
			})

			// Update the deployment order
			f.fuzzer.config.Fuzzing.DeploymentOrder = []string{"InheritedSecondContract", "InheritedFirstContract"}

			// Note that the fuzzer won't spin up any workers or fuzz anything. We just want to test that the coverage
			// maps don't populate due to deployment order changes
			err = f.fuzzer.Start()
			assert.NoError(t, err)

			// Check to see if original and new coverage are the same
			newCoverage := f.fuzzer.corpus.CoverageMaps()
			assert.False(t, originalCoverage.Equal(newCoverage))
		},
	})
}
