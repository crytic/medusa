package fuzzing

import (
	"github.com/trailofbits/medusa/chain"
	"github.com/trailofbits/medusa/events"
	"github.com/trailofbits/medusa/fuzzing/types"
	"github.com/trailofbits/medusa/fuzzing/valuegeneration"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/fuzzing/config"
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
			existingValueGenFunc := f.fuzzer.Hooks.NewValueGeneratorFunc
			f.fuzzer.Hooks.NewValueGeneratorFunc = func(fuzzer *Fuzzer, valueSet *valuegeneration.ValueSet) (valuegeneration.ValueGenerator, error) {
				valueGenOk = true
				return existingValueGenFunc(fuzzer, valueSet)
			}
			existingChainSetupFunc := f.fuzzer.Hooks.ChainSetupFunc
			f.fuzzer.Hooks.ChainSetupFunc = func(fuzzer *Fuzzer, testChain *chain.TestChain) error {
				chainSetupOk = true
				return existingChainSetupFunc(fuzzer, testChain)
			}
			f.fuzzer.Hooks.CallSequenceTestFuncs = append(f.fuzzer.Hooks.CallSequenceTestFuncs, func(worker *FuzzerWorker, callSequence types.CallSequence) ([]ShrinkCallSequenceRequest, error) {
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

// TestAssertionsBasicSolving runs tests to ensure that assertion testing behaves as expected.
func TestAssertionsBasicSolving(t *testing.T) {
	filePaths := []string{
		"testdata/contracts/assertions/assert_immediate.sol",
		"testdata/contracts/assertions/assert_even_number.sol",
	}
	for _, filePath := range filePaths {
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
			config.Fuzzing.TransactionGasLimit = 100000                          // we set this low, so contract execution runs out of gas earlier.
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
			filePath:    filePath,
			solcVersion: "0.7.0", // this test depends on solc <0.8.0
			configUpdates: func(config *config.ProjectConfig) {
				config.Fuzzing.DeploymentOrder = []string{"InnerDeploymentFactory"}
				config.Fuzzing.TestLimit = 500 // this test should expose a failure quickly.
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

// TestInitializeCoverageMaps will test whether the corpus can be "replayed" to seed the fuzzer with coverage from a
// previous run.
func TestInitializeCoverageMaps(t *testing.T) {
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
			originalCoverage := f.fuzzer.coverageMaps

			// Subscribe to the event and stop the fuzzer
			f.fuzzer.Events.FuzzerStarting.Subscribe(func(event FuzzerStartingEvent) error {
				// Simply stop the fuzzer
				event.Fuzzer.Stop()
				return nil
			})

			// Note that the fuzzer won't spin up any workers or fuzz anything. We just want to test that we seeded
			// the coverage maps properly
			err = f.fuzzer.Start()
			assert.NoError(t, err)

			// Check to see if we have some coverage
			assertCorpusCallSequencesCollected(f, true)
			newCoverage := f.fuzzer.coverageMaps

			// Check to see if original and new coverage are the same
			assert.True(t, originalCoverage.Equals(newCoverage))
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
			originalCoverage := f.fuzzer.coverageMaps

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
			newCoverage := f.fuzzer.coverageMaps
			assert.False(t, originalCoverage.Equals(newCoverage))
		},
	})
}
