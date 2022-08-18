package fuzzing

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/compilation"
	"github.com/trailofbits/medusa/compilation/platforms"
	"github.com/trailofbits/medusa/configs"
	"github.com/trailofbits/medusa/utils/test_utils"
	"testing"
)

// FuzzSolcTarget copies a given solidity file to a temporary test directory, compiles it, and runs the fuzzer
// against it. It asserts that the fuzzer should find a result prior to timeout/cancellation.
func testFuzzSolcTarget(t *testing.T, solidityFile string, fuzzingConfig configs.FuzzingConfig) {
	// Print a status message
	fmt.Printf("Fuzzing to solve '%s'...", solidityFile)

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
	assert.True(t, len(fuzzer.Results().GetFailedTests()) > 0)
}

// FuzzSolcTargets copies the given solidity files to a temporary test directory, compiles them, and runs the fuzzer
// against them. It asserts that the fuzzer should find a result prior to timeout/cancellation for each test.
func testFuzzSolcTargets(t *testing.T, solidityFiles []string, fuzzingConfig configs.FuzzingConfig) {
	// For each solidity file, we invoke fuzzing.
	for _, solidityFile := range solidityFiles {
		testFuzzSolcTarget(t, solidityFile, fuzzingConfig)
	}
}

// TestFuzzMagicNumbers tests files runs tests against smart contracts which make use of magic numbers.
func TestFuzzMagicNumbers(t *testing.T) {
	// Create our configuration for this fuzzing campaign
	fuzzConfig := configs.FuzzingConfig{
		Workers:                  10,
		WorkerDatabaseEntryLimit: 1000,
		Timeout:                  30,
		MaxTxSequenceLength:      10,
	}
	// Copy our set of contracts to test
	testContracts := []string{
		"testdata/contracts/magic_numbers/simple_xy.sol",
		"testdata/contracts/magic_numbers/simple_xy_payable.sol",
	}

	// Fuzz all contracts
	testFuzzSolcTargets(t, testContracts, fuzzConfig)
}
