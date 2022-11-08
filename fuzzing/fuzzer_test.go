package fuzzing

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/compilation"
	"github.com/trailofbits/medusa/compilation/platforms"
	"github.com/trailofbits/medusa/fuzzing/config"
	"github.com/trailofbits/medusa/utils/testutils"
)

// TODO: Feel free to delete
// FuzzSolcTarget copies a given solidity file to a temporary test directory, compiles it, and runs the fuzzer
// against it. It asserts that the fuzzer should find a result prior to timeout/cancellation.
func testFuzzSolcTarget(t *testing.T, solidityFile string, fuzzingConfig *config.FuzzingConfig, expectFailure bool, stop bool) *Fuzzer {
	// Print a status message
	fmt.Printf("##############################################################\n")
	fmt.Printf("Fuzzing '%s'...\n", solidityFile)
	fmt.Printf("##############################################################\n")

	// Copy our target file to our test directory
	testContractPath := testutils.CopyToTestDirectory(t, solidityFile)
	// Declare the fuzzer here so that we can return a pointer to it at the end of the function
	var fuzzer *Fuzzer

	// Run the test in our temporary test directory to avoid artifact pollution.
	testutils.ExecuteInDirectory(t, testContractPath, func() {
		// Create a default solc platform config
		solcPlatformConfig := platforms.NewSolcCompilationConfig(testContractPath)

		// Wrap the platform config in a compilation config
		compilationConfig, err := compilation.NewCompilationConfigFromPlatformConfig(solcPlatformConfig)
		assert.NoError(t, err)

		// Create a project configuration to run the fuzzer with
		projectConfig := &config.ProjectConfig{
			Fuzzing:     *fuzzingConfig,
			Compilation: compilationConfig,
		}

		// Create a fuzzer instance if one has not been provided
		fuzzer, err = NewFuzzer(*projectConfig)
		assert.NoError(t, err)
		// Subscribe to `OnFuzzerStarting` event
		onFuzzerStartingCount := 0
		fuzzer.OnFuzzerStartingEventEmitter.Subscribe(func(event OnFuzzerStarting) {
			onFuzzerStartingCount += 1
			// Immediately shutdown the fuzzer if requested. This is good for unit testing actions done before starting
			// the fuzzing loop
			if stop {
				fuzzer.Stop()
				// This assertion is proof that we did not create any workers
				assert.True(t, fuzzer.workers == nil)
			}
		})
		// Run the fuzzer against the compilation
		err = fuzzer.Start()
		assert.NoError(t, err)
		// Break out of asserting anything else in this test function
		if stop {
			return
		}
		// Assert that we captured the OnFuzzerStarting event
		assert.True(t, onFuzzerStartingCount == 1)
		// Ensure we captured a failed test.
		if expectFailure {
			assert.True(t, len(fuzzer.TestCasesWithStatus(TestCaseStatusFailed)) > 0, "Fuzz test could not be solved before timeout ("+strconv.Itoa(projectConfig.Fuzzing.Timeout)+" seconds)")
		} else {
			assert.True(t, len(fuzzer.TestCasesWithStatus(TestCaseStatusFailed)) == 0, "Fuzz test found a violated property test when it should not have")
		}
		// If default configuration is used, all test contracts should show some level of coverage
		if fuzzingConfig.CoverageEnabled {
			assert.True(t, fuzzer.corpus.CallSequenceCount() > 0, "No coverage was captured")
		}
	})
	return fuzzer
}

// TestDeploymentInnerDeployment runs a test to ensure dynamically deployed contracts are detected by the Fuzzer and
// their properties are tested appropriately. This test contract deploys the inner contract which takes no constructor
// arguments
func TestDeploymentInnerDeployment(t *testing.T) {
	// Define target and filename
	target := "testdata/contracts/deployment_tests/inner_deployment.sol"
	filename := "inner_deployment.sol"
	compilationConfig, err := getSolcCompilationConfig(filename)
	assert.NoError(t, err)
	projectConfig := getDefaultProjectConfig(compilationConfig)
	// Set up the fuzzer environment
	setupFuzzerEnvironment(t, projectConfig, target, func(fctx *fuzzerTestingContext) {
		fuzzer := fctx.fuzzer
		// Setup checks for event emissions
		expectEmittedEvent(t, fctx, &fuzzer.OnFuzzerStartingEventEmitter)
		// Start the fuzzer
		err = fuzzer.Start()
		assert.NoError(t, err)
		// Check to see if there are any failures and make sure there is some coverage
		assertFailedTestExists(t, fuzzer, true)
		assertCoverageCollected(t, fuzzer, true)
	})
}

// TestDeploymentInnerDeploymentOnConstruction runs a test to ensure dynamically deployed contracts are detected by the
// Fuzzer and their properties are tested appropriately. This test contract deploys the inner contract which takes
// constructor arguments, at construction time.
func TestDeploymentInnerDeploymentOnConstruction(t *testing.T) {
	// Define target and filename
	target := "testdata/contracts/deployment_tests/inner_deployment_on_construction.sol"
	filename := "inner_deployment_on_construction.sol"
	compilationConfig, err := getSolcCompilationConfig(filename)
	assert.NoError(t, err)
	projectConfig := getDefaultProjectConfig(compilationConfig)
	// Set up the fuzzer environment
	setupFuzzerEnvironment(t, projectConfig, target, func(fctx *fuzzerTestingContext) {
		fuzzer := fctx.fuzzer
		// Setup checks for event emissions
		expectEmittedEvent(t, fctx, &fuzzer.OnFuzzerStartingEventEmitter)
		// Start the fuzzer
		err = fuzzer.Start()
		assert.NoError(t, err)
		// Check to see if there are any failures and make sure there is some coverage
		assertFailedTestExists(t, fuzzer, true)
		assertCoverageCollected(t, fuzzer, true)
	})
}

// TestDeploymentInnerDeploymentOnConstruction runs a test to ensure dynamically deployed contracts are detected by the
// Fuzzer and their properties are tested appropriately. This test contract deploys the inner contract which takes
// constructor arguments, during the fuzzing campaign.
func TestDeploymentInnerInnerDeployment(t *testing.T) {
	// Define target and filename
	target := "testdata/contracts/deployment_tests/inner_inner_deployment.sol"
	filename := "inner_inner_deployment.sol"
	compilationConfig, err := getSolcCompilationConfig(filename)
	assert.NoError(t, err)
	projectConfig := getDefaultProjectConfig(compilationConfig)
	// Set up the fuzzer environment
	setupFuzzerEnvironment(t, projectConfig, target, func(fctx *fuzzerTestingContext) {
		fuzzer := fctx.fuzzer
		// Setup checks for event emissions
		expectEmittedEvent(t, fctx, &fuzzer.OnFuzzerStartingEventEmitter)
		// Start the fuzzer
		err = fuzzer.Start()
		assert.NoError(t, err)
		// Check to see if there are any failures and make sure there is some coverage
		assertFailedTestExists(t, fuzzer, true)
		assertCoverageCollected(t, fuzzer, true)
	})
}

// TestDeploymentInternalLibrary runs a test to ensure internal libraries behave correctly.
func TestDeploymentInternalLibrary(t *testing.T) {
	// Define target and filename
	target := "testdata/contracts/deployment_tests/internal_library.sol"
	filename := "internal_library.sol"
	// Update timeout and test limit to make the testing cycle shorter
	fuzzConfig := getDefaultFuzzingConfig()
	fuzzConfig.Timeout = 10
	fuzzConfig.TestLimit = 10000
	compilationConfig, err := getSolcCompilationConfig(filename)
	assert.NoError(t, err)
	projectConfig := getProjectConfig(fuzzConfig, compilationConfig)
	// Set up the fuzzer environment
	setupFuzzerEnvironment(t, projectConfig, target, func(fctx *fuzzerTestingContext) {
		fuzzer := fctx.fuzzer
		// Setup checks for event emissions
		expectEmittedEvent(t, fctx, &fuzzer.OnFuzzerStartingEventEmitter)
		// Start the fuzzer
		err = fuzzer.Start()
		assert.NoError(t, err)
		// Check to see if there are any failures and make sure there is some coverage
		assertFailedTestExists(t, fuzzer, false)
		assertCoverageCollected(t, fuzzer, true)
	})
}

// TestFuzzMagicNumbersSimpleXY runs a test to solve specific function input parameters.
func TestFuzzMagicNumbersSimpleXY(t *testing.T) {
	// Define target and filename
	target := "testdata/contracts/magic_numbers/simple_xy.sol"
	filename := "simple_xy.sol"
	compilationConfig, err := getSolcCompilationConfig(filename)
	assert.NoError(t, err)
	projectConfig := getDefaultProjectConfig(compilationConfig)
	// Set up the fuzzer environment
	setupFuzzerEnvironment(t, projectConfig, target, func(fctx *fuzzerTestingContext) {
		fuzzer := fctx.fuzzer
		// Setup checks for event emissions
		expectEmittedEvent(t, fctx, &fuzzer.OnFuzzerStartingEventEmitter)
		// Start the fuzzer
		err = fuzzer.Start()
		assert.NoError(t, err)
		// Check to see if there are any failures and make sure there is some coverage
		assertFailedTestExists(t, fuzzer, true)
		assertCoverageCollected(t, fuzzer, true)
	})
}

// TestFuzzMagicNumbersSimpleXYPayable runs a test to solve specific payable values.
func TestFuzzMagicNumbersSimpleXYPayable(t *testing.T) {
	// Define target and filename
	target := "testdata/contracts/magic_numbers/simple_xy_payable.sol"
	filename := "simple_xy_payable.sol"
	compilationConfig, err := getSolcCompilationConfig(filename)
	assert.NoError(t, err)
	projectConfig := getDefaultProjectConfig(compilationConfig)
	// Set up the fuzzer environment
	setupFuzzerEnvironment(t, projectConfig, target, func(fctx *fuzzerTestingContext) {
		fuzzer := fctx.fuzzer
		// Setup checks for event emissions
		expectEmittedEvent(t, fctx, &fuzzer.OnFuzzerStartingEventEmitter)
		// Start the fuzzer
		err = fuzzer.Start()
		assert.NoError(t, err)
		// Check to see if there are any failures and make sure there is some coverage
		assertFailedTestExists(t, fuzzer, true)
		assertCoverageCollected(t, fuzzer, true)
	})
}

// TestFuzzVMBlockNumber runs a test to ensure block numbers behave correctly in the VM.
func TestFuzzVMBlockNumber(t *testing.T) {
	// Define target and filename
	target := "testdata/contracts/vm_tests/block_number.sol"
	filename := "block_number.sol"
	compilationConfig, err := getSolcCompilationConfig(filename)
	assert.NoError(t, err)
	projectConfig := getDefaultProjectConfig(compilationConfig)
	// Set up the fuzzer environment
	setupFuzzerEnvironment(t, projectConfig, target, func(fctx *fuzzerTestingContext) {
		fuzzer := fctx.fuzzer
		// Setup checks for event emissions
		expectEmittedEvent(t, fctx, &fuzzer.OnFuzzerStartingEventEmitter)
		// Start the fuzzer
		err = fuzzer.Start()
		assert.NoError(t, err)
		// Check to see if there are any failures and make sure there is some coverage
		assertFailedTestExists(t, fuzzer, true)
		assertCoverageCollected(t, fuzzer, true)
	})
}

// TestFuzzVMTimestamp runs a test to ensure block timestamps behave correctly in the VM.
func TestFuzzVMTimestamp(t *testing.T) {
	// Define target and filename
	target := "testdata/contracts/vm_tests/block_timestamp.sol"
	filename := "block_timestamp.sol"
	compilationConfig, err := getSolcCompilationConfig(filename)
	assert.NoError(t, err)
	projectConfig := getDefaultProjectConfig(compilationConfig)
	// Set up the fuzzer environment
	setupFuzzerEnvironment(t, projectConfig, target, func(fctx *fuzzerTestingContext) {
		fuzzer := fctx.fuzzer
		// Setup checks for event emissions
		expectEmittedEvent(t, fctx, &fuzzer.OnFuzzerStartingEventEmitter)
		// Start the fuzzer
		err = fuzzer.Start()
		assert.NoError(t, err)
		// Check to see if there are any failures and make sure there is some coverage
		assertFailedTestExists(t, fuzzer, true)
		assertCoverageCollected(t, fuzzer, true)
	})
}

// TestFuzzVMBlockHash runs a test to ensure block hashes behave correctly in the VM.
func TestFuzzVMBlockHash(t *testing.T) {
	// Define target and filename
	target := "testdata/contracts/vm_tests/block_hash.sol"
	filename := "block_hash.sol"
	// Update timeout and test limit to make the testing cycle shorter
	fuzzConfig := getDefaultFuzzingConfig()
	fuzzConfig.Timeout = 10
	fuzzConfig.TestLimit = 10000
	compilationConfig, err := getSolcCompilationConfig(filename)
	assert.NoError(t, err)
	projectConfig := getProjectConfig(fuzzConfig, compilationConfig)
	// Set up the fuzzer environment
	setupFuzzerEnvironment(t, projectConfig, target, func(fctx *fuzzerTestingContext) {
		fuzzer := fctx.fuzzer
		// Setup checks for event emissions
		expectEmittedEvent(t, fctx, &fuzzer.OnFuzzerStartingEventEmitter)
		// Start the fuzzer
		err = fuzzer.Start()
		assert.NoError(t, err)
		// Check to see if there are any failures and make sure there is some coverage
		assertFailedTestExists(t, fuzzer, false)
		assertCoverageCollected(t, fuzzer, true)
	})
}

// TestInitializeCoverageMaps will test whether the corpus can be "replayed" to seed the fuzzer with coverage from a
// previous run.
func TestInitializeCoverageMaps(t *testing.T) {
	// Define target and filename
	target := "testdata/contracts/magic_numbers/simple_xy.sol"
	filename := "simple_xy.sol"
	compilationConfig, err := getSolcCompilationConfig(filename)
	assert.NoError(t, err)
	projectConfig := getDefaultProjectConfig(compilationConfig)
	// Set up the fuzzer environment
	setupFuzzerEnvironment(t, projectConfig, target, func(fctx *fuzzerTestingContext) {
		fuzzer := fctx.fuzzer
		// Setup checks for event emissions
		expectEmittedEvent(t, fctx, &fuzzer.OnFuzzerStartingEventEmitter)
		// Start the fuzzer
		err = fuzzer.Start()
		assert.NoError(t, err)
		// Make sure we have some coverage
		assertCoverageCollected(t, fuzzer, true)
		// Cache current coverage maps
		originalCoverage := fuzzer.corpus.CoverageMaps()
		// Subscribe to the event and stop the fuzzer
		fuzzer.OnFuzzerStartingEventEmitter.Subscribe(stopFuzzerOnFuzzerStartingEvent)
		// Note that the fuzzer won't spin up any workers or fuzz anything. We just want to test that we seeded
		// the coverage maps properly
		err = fuzzer.Start()
		assert.NoError(t, err)
		// Check to see if we have some coverage
		assertCoverageCollected(t, fuzzer, true)
		newCoverage := fuzzer.corpus.CoverageMaps()
		// Check to see if original and new coverage are the same
		assert.True(t, originalCoverage.Equals(newCoverage))
	})
}

// TestDeploymentOrderWithCoverage will ensure that changing the deployment order does not lead to the same coverage
// This is also proof that changing the order changes the addresses of the contracts leading to the coverage not being
// useful.
func TestDeploymentOrderWithCoverage(t *testing.T) {
	// Define target and directory
	// We want to deploy a whole project with multiple contracts so we are using a hardhat project
	// TODO: There are now two versions of basic_project. One for compilation and one for fuzzing. Is there a better
	//  route here?
	target := "testdata/hardhat/basic_project/"
	directory := "."
	compilationConfig, err := getCryticCompileCompilationConfig(directory)
	assert.NoError(t, err)
	fuzzConfig := getDefaultFuzzingConfig()
	// Set deployment order
	fuzzConfig.DeploymentOrder = []string{"InheritedFirstContract", "InheritedSecondContract"}
	projectConfig := getProjectConfig(fuzzConfig, compilationConfig)
	// Provide "npm install" command to run
	command := []string{"npm", "install"}
	// Set up the fuzzer environment
	setupFuzzerEnvironment(t, projectConfig, target, func(fctx *fuzzerTestingContext) {
		fuzzer := fctx.fuzzer
		// Setup checks for event emissions
		expectEmittedEvent(t, fctx, &fuzzer.OnFuzzerStartingEventEmitter)
		// Start the fuzzer
		err = fuzzer.Start()
		assert.NoError(t, err)
		// Make sure we have some coverage
		assertCoverageCollected(t, fuzzer, true)
		// Cache current coverage maps
		originalCoverage := fuzzer.corpus.CoverageMaps()
		// Subscribe to the event and stop the fuzzer
		fuzzer.OnFuzzerStartingEventEmitter.Subscribe(stopFuzzerOnFuzzerStartingEvent)
		// Update the deployment order
		fuzzer.config.Fuzzing.DeploymentOrder = []string{"InheritedSecondContract", "InheritedFirstContract"}
		// Note that the fuzzer won't spin up any workers or fuzz anything. We just want to test that the coverage
		// maps don't populate due to deployment order changes
		err = fuzzer.Start()
		assert.NoError(t, err)
		newCoverage := fuzzer.corpus.CoverageMaps()
		// Check to see if original and new coverage are the same
		assert.False(t, originalCoverage.Equals(newCoverage))
	}, command)
}

// TODO: Does this make sense? Specifically, the targetPathInfo.Name() part. Wanted to avoid parsing the string directly
// testFuzzSolcTargets copies the given solidity files to a temporary test directory, compiles them, and runs the fuzzer
// against them. It asserts that the fuzzer should find a result prior to timeout/cancellation for each test.
func testFuzzSolcTargets(t *testing.T, targets []string, fuzzingConfig *config.FuzzingConfig, expectFailure bool) {
	// For each solidity file, we invoke fuzzing.
	for _, target := range targets {
		// Get the filename given a target path
		targetPathInfo, err := os.Stat(target)
		assert.NoError(t, err)
		filename := targetPathInfo.Name()
		compilationConfig, err := getSolcCompilationConfig(filename)
		assert.NoError(t, err)
		projectConfig := getProjectConfig(fuzzingConfig, compilationConfig)
		// Set up the fuzzer environment
		setupFuzzerEnvironment(t, projectConfig, target, func(fctx *fuzzerTestingContext) {
			fuzzer := fctx.fuzzer
			// Setup checks for event emissions
			expectEmittedEvent(t, fctx, &fuzzer.OnFuzzerStartingEventEmitter)
			// Start the fuzzer
			err = fuzzer.Start()
			assert.NoError(t, err)
			// Check to see if there are any failures and make sure there is some coverage
			assertFailedTestExists(t, fuzzer, expectFailure)
			assertCoverageCollected(t, fuzzer, true)
		})
	}
}
