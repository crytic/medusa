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

// FuzzSolcTarget copies a given solidity file to a temporary test directory, compiles it, and runs the fuzzer
// against it. It asserts that the fuzzer should find a result prior to timeout/cancellation.
func testFuzzSolcTarget(t *testing.T, solidityFile string, fuzzingConfig configs.FuzzingConfig) {
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
		Fuzzing:     fuzzingConfig,
		Compilation: *compilationConfig,
	}

	// Create a fuzzer instance
	fuzzer, err := NewFuzzer(*projectConfig)
	assert.Nil(t, err)

	// Run the fuzzer against the compilation
	err = fuzzer.Start()
	assert.Nil(t, err)

	// Ensure we captured a failed test.
	assert.True(t, len(fuzzer.Results().GetFailedTests()) > 0, "Fuzz test could not be solved before timeout ("+strconv.Itoa(projectConfig.Fuzzing.Timeout)+" seconds)")
}

// FuzzSolcTargets copies the given solidity files to a temporary test directory, compiles them, and runs the fuzzer
// against them. It asserts that the fuzzer should find a result prior to timeout/cancellation for each test.
func testFuzzSolcTargets(t *testing.T, solidityFiles []string, fuzzingConfig configs.FuzzingConfig) {
	// For each solidity file, we invoke fuzzing.
	for _, solidityFile := range solidityFiles {
		testFuzzSolcTarget(t, solidityFile, fuzzingConfig)
	}
}

// Create our configuration for testing
var defaultTestingFuzzConfig = configs.FuzzingConfig{
	Workers:                  10,
	WorkerDatabaseEntryLimit: 1000,
	Timeout:                  30,
	TestLimit:                0,
	MaxTxSequenceLength:      100,
	TestPrefixes: []string{
		"fuzz_", "echidna_",
	},
}

// TestFuzzMagicNumbersSimpleXY runs a test to solve simple_xy.sol
func TestFuzzMagicNumbersSimpleXY(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/magic_numbers/simple_xy.sol", defaultTestingFuzzConfig)
}

// TestFuzzMagicNumbersSimpleXYPayable runs a test to solve simple_xy_payable.sol
func TestFuzzMagicNumbersSimpleXYPayable(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/magic_numbers/simple_xy_payable.sol", defaultTestingFuzzConfig)
}
