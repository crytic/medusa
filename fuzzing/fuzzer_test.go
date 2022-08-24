package fuzzing

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/compilation"
	"github.com/trailofbits/medusa/compilation/platforms"
	"github.com/trailofbits/medusa/configs"
	"github.com/trailofbits/medusa/fuzzing/types"
	"github.com/trailofbits/medusa/utils/test_utils"
	"os"
	"path/filepath"
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
		Coverage:        true,
		CorpusDirectory: "corpus",
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

	test_utils.ExecuteInDirectory(t, testContractPath, func() {
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
		// If default configuration is used, all test contracts should show some level of coverage
		if fuzzingConfig.Coverage {
			assert.True(t, len(fuzzer.corpus.CorpusBlockSequences) > 0, "No coverage was captured")
			if fuzzingConfig.CorpusDirectory != "" {
				testCoverageMarshalingAndUnmarsharling(t, fuzzer)
			}
		}
	})
}

// testCoverageMarshalingAndUnmarsharling tests to make sure that a marshaled and then unmarshaled
// CorpusBlockSequence preserves the sequence.
func testCoverageMarshalingAndUnmarsharling(t *testing.T, fuzzer *Fuzzer) {
	// Get current working directory
	currentDir, _ := os.Getwd()
	// Iterate through each sequence in the corpus
	for hash, seqOne := range fuzzer.corpus.CorpusBlockSequences {
		// Move to corpus/corpus subdirectory
		os.Chdir(filepath.Join(currentDir, fuzzer.config.Fuzzing.CorpusDirectory, "/corpus"))
		// Read corpus file
		b, _ := os.ReadFile(hash + ".json")
		// Unmarshal corpus file
		var seqTwo types.CorpusBlockSequence
		err := json.Unmarshal(b, &seqTwo)
		if err != nil {
			assert.False(t, false, "JSON unmarshaling failed")
		}
		// Assert that the unmarshaled sequence is equal to the sequence (which was marshaled to begin with)
		assert.True(t, test_utils.SequencesAreEqual(t, *seqOne, seqTwo), "marshaling and unmarshaling are not symmetric operations")
	}
}

// testFuzzSolcTargets copies the given solidity files to a temporary test directory, compiles them, and runs the fuzzer
// against them. It asserts that the fuzzer should find a result prior to timeout/cancellation for each test.
func testFuzzSolcTargets(t *testing.T, solidityFiles []string, fuzzingConfig *configs.FuzzingConfig, expectFailure bool) {
	// For each solidity file, we invoke fuzzing.
	for _, solidityFile := range solidityFiles {
		testFuzzSolcTarget(t, solidityFile, fuzzingConfig, expectFailure)
	}
}

// TestFuzzVMBlockNumber runs a test to solve block_number.sol
func TestFuzzVMBlockNumber(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/vm_tests/block_number.sol", getFuzzConfigDefault(), true)
}

// TestFuzzVMTimestamp runs a test to solve block_number.sol
func TestFuzzVMTimestamp(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/vm_tests/block_timestamp.sol", getFuzzConfigDefault(), true)
}

// TestFuzzVMBlockHash runs a test to solve block_hash.sol
func TestFuzzVMBlockHash(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/vm_tests/block_hash.sol", getFuzzConfigCantSolveShortTime(), false)
}

// TestFuzzMagicNumbersSimpleXY runs a test to solve simple_xy.sol
func TestFuzzMagicNumbersSimpleXY(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/magic_numbers/simple_xy.sol", getFuzzConfigDefault(), true)
}

// TestFuzzMagicNumbersSimpleXYPayable runs a test to solve simple_xy_payable.sol
func TestFuzzMagicNumbersSimpleXYPayable(t *testing.T) {
	testFuzzSolcTarget(t, "testdata/contracts/magic_numbers/simple_xy_payable.sol", getFuzzConfigDefault(), true)
}
