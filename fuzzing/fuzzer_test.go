package fuzzing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/fuzzing/config"
)

// TestDeploymentInnerDeployment runs a test to ensure dynamically deployed contracts are detected by the Fuzzer and
// their properties are tested appropriately. This test contract deploys the inner contract by calling a method after
// deployment of the factory contract.
func TestDeploymentInnerDeployment(t *testing.T) {
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/deployment_tests/inner_deployment.sol",
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

// TestDeploymentInnerDeploymentOnConstruction runs a test to ensure dynamically deployed contracts are detected by the
// Fuzzer and their properties are tested appropriately. This test contract deploys the inner contract during
// construction of the factory contract.
func TestDeploymentInnerDeploymentOnConstruction(t *testing.T) {
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/deployment_tests/inner_deployment_on_construction.sol",
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

// TestDeploymentInnerDeploymentOnConstruction runs a test to ensure dynamically deployed contracts are detected by the
// Fuzzer and their properties are tested appropriately. This test contract deploys the inner contract which takes
// constructor arguments, during the fuzzing campaign.
func TestDeploymentInnerInnerDeployment(t *testing.T) {
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/deployment_tests/inner_inner_deployment.sol",
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

// TestDeploymentInternalLibrary runs a test to ensure internal libraries behave correctly.
func TestDeploymentInternalLibrary(t *testing.T) {
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/deployment_tests/internal_library.sol",
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

// TestFuzzValueGenerationSolving runs a series of tests to test the value generator can solve expected problems.
func TestFuzzValueGenerationSolving(t *testing.T) {
	filePaths := []string{
		"testdata/contracts/value_generation/match_addr_contract.sol",
		"testdata/contracts/value_generation/match_addr_exact.sol",
		"testdata/contracts/value_generation/match_addr_sender.sol",
		"testdata/contracts/value_generation/match_string_exact.sol",
		"testdata/contracts/value_generation/match_structs_xy.sol",
		//"testdata/contracts/value_generation/match_ints_xy.sol", // TODO: Investigate why this is much less slow to solve than match_uints_xy in the value generator.
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

// TestFuzzVMBlockNumber runs a test to ensure block numbers behave correctly in the VM.
func TestFuzzVMBlockNumber(t *testing.T) {
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/vm_tests/block_number.sol",
		configUpdates: func(config *config.ProjectConfig) {
			config.Fuzzing.DeploymentOrder = []string{"TestBlockNumber"}
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

// TestFuzzVMTimestamp runs a test to ensure block timestamps behave correctly in the VM.
func TestFuzzVMTimestamp(t *testing.T) {
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/vm_tests/block_timestamp.sol",
		configUpdates: func(config *config.ProjectConfig) {
			config.Fuzzing.DeploymentOrder = []string{"TestBlockTimestamp"}
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

// TestFuzzVMBlockHash runs a test to ensure block hashes behave correctly in the VM.
func TestFuzzVMBlockHash(t *testing.T) {
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/vm_tests/block_hash.sol",
		configUpdates: func(config *config.ProjectConfig) {
			config.Fuzzing.DeploymentOrder = []string{"TestBlockHash"}
			config.Fuzzing.TestLimit = 1_000 // this test should expose a failure quickly.
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
		},
		method: func(f *fuzzerTestContext) {
			// Setup checks for event emissions
			expectEventEmitted(f, &f.fuzzer.OnStartingEventEmitter)

			// Start the fuzzer
			err := f.fuzzer.Start()
			assert.NoError(t, err)

			// Make sure we have some coverage
			assertCorpusCallSequencesCollected(f, true)

			// Cache current coverage maps
			originalCoverage := f.fuzzer.coverageMaps

			// Subscribe to the event and stop the fuzzer
			f.fuzzer.OnStartingEventEmitter.Subscribe(func(event OnFuzzerStarting) error {
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
		filePath: "testdata/contracts/deployment_tests/deployment_order.sol",
		configUpdates: func(config *config.ProjectConfig) {
			config.Fuzzing.DeploymentOrder = []string{"InheritedFirstContract", "InheritedSecondContract"}
		},
		method: func(f *fuzzerTestContext) {
			// Setup checks for event emissions
			expectEventEmitted(f, &f.fuzzer.OnStartingEventEmitter)

			// Start the fuzzer
			err := f.fuzzer.Start()
			assert.NoError(t, err)

			// Make sure we have some coverage
			assertCorpusCallSequencesCollected(f, true)

			// Cache current coverage maps
			originalCoverage := f.fuzzer.coverageMaps

			// Subscribe to the event and stop the fuzzer
			f.fuzzer.OnStartingEventEmitter.Subscribe(func(event OnFuzzerStarting) error {
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
