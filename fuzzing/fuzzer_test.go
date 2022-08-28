package fuzzing

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/compilation"
	"github.com/trailofbits/medusa/compilation/platforms"
	"github.com/trailofbits/medusa/configs"
	"github.com/trailofbits/medusa/utils/test_utils"
	"strconv"
	"testing"
)

// getFuzzConfigDefault obtains the default configuration for tests.
func getFuzzConfigDefault() *configs.FuzzingConfig {
	return &configs.FuzzingConfig{
		Workers:                  10,
		WorkerDatabaseEntryLimit: 10000,
		Timeout:                  30,
		TestLimit:                0,
		MaxTxSequenceLength:      100,
		TestPrefixes: []string{
			"fuzz_", "echidna_",
		},
	}
}

// getFuzzConfigCantSolveShortTime obtains a configuration for tests which we expect to run for a few seconds without
// triggering an error.
func getFuzzConfigCantSolveShortTime() *configs.FuzzingConfig {
	config := getFuzzConfigDefault()
	config.Timeout = 10
	config.TestLimit = 10000
	return config
}

// FuzzSolcTarget copies a given solidity file to a temporary test directory, compiles it, and runs the fuzzer
// against it. It asserts that the fuzzer should find a result prior to timeout/cancellation.
func testFuzzSolcTarget(t *testing.T, solidityFile string, fuzzingConfig *configs.FuzzingConfig, expectFailure bool) {
	// Print a status message
	fmt.Printf("##############################################################\n")
	fmt.Printf("Fuzzing '%s'...\n", solidityFile)
	fmt.Printf("##############################################################\n")

	// Copy our target file to our test directory
	testContractPath := test_utils.CopyToTestDirectory(t, solidityFile)

	// Create a default solc platform config
	solcPlatformConfig := platforms.NewSolcCompilationConfig(testContractPath)

	// Wrap the platform config in a compilation config
	compilationConfig, err := compilation.GetCompilationConfigFromPlatformConfig(solcPlatformConfig)
	assert.Nil(t, err)

	// Create a project configuration to run the fuzzer with
	projectConfig := &configs.ProjectConfig{
		Accounts: configs.AccountConfig{
			Generate: 5,
		},
		Fuzzing:     *fuzzingConfig,
		Compilation: *compilationConfig,
	}

	// Create a fuzzer instance
	fuzzer, err := NewFuzzer(*projectConfig)
	assert.Nil(t, err)

	// Run the fuzzer against the compilation
	err = fuzzer.Start()
	assert.Nil(t, err)

	// Ensure we captured a failed test.
	if expectFailure {
		assert.True(t, len(fuzzer.Results().GetFailedTests()) > 0, "Fuzz test could not be solved before timeout ("+strconv.Itoa(projectConfig.Fuzzing.Timeout)+" seconds)")
	} else {
		assert.True(t, len(fuzzer.Results().GetFailedTests()) == 0, "Fuzz test found a violated property test when it should not have")
	}
}

// FuzzSolcTargets copies the given solidity files to a temporary test directory, compiles them, and runs the fuzzer
// against them. It asserts that the fuzzer should find a result prior to timeout/cancellation for each test.
func testFuzzSolcTargets(t *testing.T, solidityFiles []string, fuzzingConfig *configs.FuzzingConfig, expectFailure bool) {
	// For each solidity file, we invoke fuzzing.
	for _, solidityFile := range solidityFiles {
		testFuzzSolcTarget(t, solidityFile, fuzzingConfig, expectFailure)
	}
}

// TestDeploymentInternalLibrary runs a test against internal_library.sol to ensure internal libraries behave correctly.
func TestDeploymentInternalLibrary(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/deployment_tests/internal_library.sol", getFuzzConfigCantSolveShortTime(), false)
}

// TestFuzzMagicNumbersSimpleXY runs a test against simple_xy.sol to solve specific function input parameters.
func TestFuzzMagicNumbersSimpleXY(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/magic_numbers/simple_xy.sol", getFuzzConfigDefault(), true)
}

// TestFuzzMagicNumbersSimpleXYPayable runs a test against simple_xy_payable.sol to solve specific payable values.
func TestFuzzMagicNumbersSimpleXYPayable(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/magic_numbers/simple_xy_payable.sol", getFuzzConfigDefault(), true)
}

// TestFuzzVMBlockNumber runs a test against block_number.sol to ensure block numbers behave correctly in the VM.
func TestFuzzVMBlockNumber(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/vm_tests/block_number.sol", getFuzzConfigDefault(), true)
}

// TestFuzzVMTimestamp runs a test against block_number.sol to ensure block timestamps behave correctly in the VM.
func TestFuzzVMTimestamp(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/vm_tests/block_timestamp.sol", getFuzzConfigDefault(), true)
}

// TestFuzzVMBlockHash runs a test against block_hash.sol to ensure block hashes behave correctly in the VM.
func TestFuzzVMBlockHash(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/vm_tests/block_hash.sol", getFuzzConfigCantSolveShortTime(), false)
}
