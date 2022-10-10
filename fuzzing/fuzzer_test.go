package fuzzing

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/compilation"
	"github.com/trailofbits/medusa/compilation/platforms"
	"github.com/trailofbits/medusa/fuzzing/config"
	"github.com/trailofbits/medusa/utils/testutils"
	"strconv"
	"testing"
)

// getFuzzConfigDefault obtains the default configuration for tests.
func getFuzzConfigDefault() *config.FuzzingConfig {
	return &config.FuzzingConfig{
		Workers:                  10,
		WorkerDatabaseEntryLimit: 10000,
		Timeout:                  30,
		TestLimit:                0,
		MaxTxSequenceLength:      100,
		SenderAddresses: []string{
			"0x1111111111111111111111111111111111111111",
			"0x2222222222222222222222222222222222222222",
			"0x3333333333333333333333333333333333333333",
		},
		DeployerAddress: "0x1111111111111111111111111111111111111111",
		Testing: config.TestingConfig{
			StopOnFailedTest: true,
			AssertionTesting: config.AssertionTestingConfig{
				Enabled:         false,
				TestViewMethods: false,
			},
			PropertyTesting: config.PropertyTestConfig{
				Enabled: true,
				TestPrefixes: []string{
					"fuzz_",
				},
			},
		},
	}
}

// getFuzzConfigCantSolveShortTime obtains a configuration for tests which we expect to run for a few seconds without
// triggering an error.
func getFuzzConfigCantSolveShortTime() *config.FuzzingConfig {
	config := getFuzzConfigDefault()
	config.Timeout = 10
	config.TestLimit = 10000
	return config
}

// FuzzSolcTarget copies a given solidity file to a temporary test directory, compiles it, and runs the fuzzer
// against it. It asserts that the fuzzer should find a result prior to timeout/cancellation.
func testFuzzSolcTarget(t *testing.T, solidityFile string, fuzzingConfig *config.FuzzingConfig, expectFailure bool) {
	// Print a status message
	fmt.Printf("##############################################################\n")
	fmt.Printf("Fuzzing '%s'...\n", solidityFile)
	fmt.Printf("##############################################################\n")

	// Copy our target file to our test directory
	testContractPath := testutils.CopyToTestDirectory(t, solidityFile)

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

	// Create a fuzzer instance
	fuzzer, err := NewFuzzer(*projectConfig)
	assert.NoError(t, err)

	// Run the fuzzer against the compilation
	err = fuzzer.Start()
	assert.NoError(t, err)

	// Ensure we captured a failed test.
	if expectFailure {
		assert.True(t, len(fuzzer.TestCasesWithStatus(TestCaseStatusFailed)) > 0, "Fuzz test could not be solved before timeout ("+strconv.Itoa(projectConfig.Fuzzing.Timeout)+" seconds)")
	} else {
		assert.True(t, len(fuzzer.TestCasesWithStatus(TestCaseStatusFailed)) == 0, "Fuzz test found a violated property test when it should not have")
	}
}

// FuzzSolcTargets copies the given solidity files to a temporary test directory, compiles them, and runs the fuzzer
// against them. It asserts that the fuzzer should find a result prior to timeout/cancellation for each test.
func testFuzzSolcTargets(t *testing.T, solidityFiles []string, fuzzingConfig *config.FuzzingConfig, expectFailure bool) {
	// For each solidity file, we invoke fuzzing.
	for _, solidityFile := range solidityFiles {
		testFuzzSolcTarget(t, solidityFile, fuzzingConfig, expectFailure)
	}
}

// TestDeploymentInnerDeployment runs a test to ensure dynamically deployed contracts are detected by the Fuzzer and
// their properties are tested appropriately. This test contract deploys the inner contract which takes no constructor
// arguments
func TestDeploymentInnerDeployment(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/deployment_tests/inner_deployment.sol", getFuzzConfigDefault(), true)
}

// TestDeploymentInnerDeploymentOnConstruction runs a test to ensure dynamically deployed contracts are detected by the
// Fuzzer and their properties are tested appropriately. This test contract deploys the inner contract which takes
// constructor arguments, at construction time.
func TestDeploymentInnerDeploymentOnConstruction(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/deployment_tests/inner_deployment_on_construction.sol", getFuzzConfigDefault(), true)
}

// TestDeploymentInnerDeploymentOnConstruction runs a test to ensure dynamically deployed contracts are detected by the
// Fuzzer and their properties are tested appropriately. This test contract deploys the inner contract which takes
// constructor arguments, during the fuzzing campaign.
func TestDeploymentInnerInnerDeployment(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/deployment_tests/inner_inner_deployment.sol", getFuzzConfigDefault(), true)
}

// TestDeploymentInternalLibrary runs a test to ensure internal libraries behave correctly.
func TestDeploymentInternalLibrary(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/deployment_tests/internal_library.sol", getFuzzConfigCantSolveShortTime(), false)
}

// TestFuzzMagicNumbersSimpleXY runs a test to solve specific function input parameters.
func TestFuzzMagicNumbersSimpleXY(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/magic_numbers/simple_xy.sol", getFuzzConfigDefault(), true)
}

// TestFuzzMagicNumbersSimpleXYPayable runs a test to solve specific payable values.
func TestFuzzMagicNumbersSimpleXYPayable(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/magic_numbers/simple_xy_payable.sol", getFuzzConfigDefault(), true)
}

// TestFuzzVMBlockNumber runs a test to ensure block numbers behave correctly in the VM.
func TestFuzzVMBlockNumber(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/vm_tests/block_number.sol", getFuzzConfigDefault(), true)
}

// TestFuzzVMTimestamp runs a test to ensure block timestamps behave correctly in the VM.
func TestFuzzVMTimestamp(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/vm_tests/block_timestamp.sol", getFuzzConfigDefault(), true)
}

// TestFuzzVMBlockHash runs a test to ensure block hashes behave correctly in the VM.
func TestFuzzVMBlockHash(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/vm_tests/block_hash.sol", getFuzzConfigCantSolveShortTime(), false)
}
