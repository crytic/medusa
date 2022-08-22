package platforms

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/compilation/types"
	"github.com/trailofbits/medusa/utils/test_utils"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testCryticGetCompiledSourceByBaseName checks if a given source file exists in a given compilation's map of sources.
// The source file is the file name of a specific file. This function simply checks one of the paths ends with
// this name. Avoid including any directories in case the path separators differ per system.
// Returns the types.CompiledSource (mapping value) associated to the path if it is found. Returns nil otherwise.
func testCryticGetCompiledSourceByBaseName(sources map[string]types.CompiledSource, name string) *types.CompiledSource {
	// Obtain a lower case version of our name to search for
	lowerName := strings.ToLower(name)

	// Search all sources for one that contains this name
	for k, v := range sources {
		// Obtain a lower case version of our path
		lowerSourcePath := strings.ToLower(k)

		// Check if our path ends with the lower name
		if strings.HasSuffix(lowerSourcePath, lowerName) {
			return &v
		}
	}

	// We did not find the source, return nil
	return nil
}

// TestCryticSingleFileNoArgsAbsolutePath tests compilation of a single with no additional arguments and absolute path
// provided.
func TestCryticSingleFileNoArgsAbsolutePath(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := test_utils.CopyToTestDirectory(t, "testdata/solc/SimpleContract.sol")

	// Execute our tests in the given test path
	test_utils.ExecuteInDirectory(t, contractPath, func() {
		// Create a cryticConfig object
		cryticConfig := NewCryticCompilationConfig(contractPath)

		// Compile the file
		compilations, _, err := cryticConfig.Compile()
		// No failures
		assert.Nil(t, err)
		// One compilation object
		assert.True(t, len(compilations) == 1)
		// One source because we specified one file
		assert.True(t, len(compilations[0].Sources) == 1)
		// Two contracts in SimpleContract.sol
		assert.True(t, len(compilations[0].Sources[contractPath].Contracts) == 2)
	})
}

// TestCryticSingleFileNoArgsRelativePath tests compilation of a single contract with no additional arguments and
// a relative path provided.
func TestCryticSingleFileNoArgsRelativePath(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := test_utils.CopyToTestDirectory(t, "testdata/solc/SimpleContract.sol")

	// Execute our tests in the given test path
	test_utils.ExecuteInDirectory(t, contractPath, func() {
		// Obtain the contract directory and cd to it.
		contractDirectory := filepath.Dir(contractPath)
		err := os.Chdir(contractDirectory)
		assert.Nil(t, err)

		// Obtain the filename
		contractName := filepath.Base(contractPath)

		// Create a cryticConfig object
		cryticConfig := NewCryticCompilationConfig(contractName)

		// Compile the file
		compilations, _, err := cryticConfig.Compile()
		// No failures
		assert.Nil(t, err)
		// One compilation object
		assert.True(t, len(compilations) == 1)
		// One source because we specified one file
		assert.True(t, len(compilations[0].Sources) == 1)
		// Two contracts in SimpleContract.sol.
		compiledSource := testCryticGetCompiledSourceByBaseName(compilations[0].Sources, contractName)
		assert.NotNil(t, compiledSource, "source file could not be resolved in compilation sources")
		assert.True(t, len(compiledSource.Contracts) == 2)
	})
}

// TestCryticSingleFileBadArgs tests compilation of a single contract with unaccepted or bad arguments
// (e.g. export-dir, export-format)
func TestCryticSingleFileBadArgs(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := test_utils.CopyToTestDirectory(t, "testdata/solc/SimpleContract.sol")

	// Execute our tests in the given test path
	test_utils.ExecuteInDirectory(t, contractPath, func() {
		// Create a crytic-compile provider
		cryticConfig := NewCryticCompilationConfig(contractPath)
		// Make sure --export-format and --export-dir are not allowed
		cryticConfig.Args = append(cryticConfig.Args, "--export-format")
		_, _, err := cryticConfig.Compile()
		// Should fail
		assert.Error(t, err)
		cryticConfig.Args = append(cryticConfig.Args, "--export-dir")
		_, _, err = cryticConfig.Compile()
		assert.Error(t, err)
		cryticConfig.Args = append(cryticConfig.Args, "--bad-arg")
		_, _, err = cryticConfig.Compile()
		assert.Error(t, err)
	})
}

// TestCryticDirectoryNoArgs tests compilation of a hardhat directory with no addition arguments provided
func TestCryticDirectoryNoArgs(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractDirectory := test_utils.CopyToTestDirectory(t, "testdata/hardhat/basic_project/")
	fmt.Printf("contract directory: %v\n", contractDirectory)

	// Execute our tests in the given test path
	test_utils.ExecuteInDirectory(t, contractDirectory, func() {
		// Run npm install
		err := exec.Command("npm", "install").Run()
		assert.Nil(t, err)

		// Create a crytic-compile provider
		cryticConfig := NewCryticCompilationConfig(contractDirectory)
		compilations, _, err := cryticConfig.Compile()
		fmt.Printf("compilations: %v\n", compilations)
		// No failures
		assert.Nil(t, err)
		fmt.Printf("number of compilations: %v\n", len(compilations))
		// Two compilation objects
		assert.True(t, len(compilations) == 2)
		// One source per compilation unit
		assert.True(t, len(compilations[0].Sources) == 1)
		assert.True(t, len(compilations[1].Sources) == 1)

		// Obtain the compiled source from both compilation units
		firstContractName := "FirstContract.sol"
		secondContractName := "SecondContract.sol"
		firstUnitFirstContractSource := testCryticGetCompiledSourceByBaseName(compilations[0].Sources, firstContractName)
		firstUnitSecondContractSource := testCryticGetCompiledSourceByBaseName(compilations[0].Sources, secondContractName)
		secondUnitFirstContractSource := testCryticGetCompiledSourceByBaseName(compilations[1].Sources, firstContractName)
		secondUnitSecondContractSource := testCryticGetCompiledSourceByBaseName(compilations[1].Sources, secondContractName)

		// Assert that each compilation unit should have two contracts in it.
		// Compilation unit ordering is non-deterministic in JSON output
		// All we care about is that each comp unit has two contracts for one or the other file
		firstCompilationUnitContractCount := 0
		if firstUnitFirstContractSource != nil {
			firstCompilationUnitContractCount += len(firstUnitFirstContractSource.Contracts)
		}
		if firstUnitSecondContractSource != nil {
			firstCompilationUnitContractCount += len(firstUnitSecondContractSource.Contracts)
		}
		assert.EqualValues(t, firstCompilationUnitContractCount, 2)

		secondCompilationUnitContractCount := 0
		if secondUnitFirstContractSource != nil {
			secondCompilationUnitContractCount += len(secondUnitFirstContractSource.Contracts)
		}
		if secondUnitSecondContractSource != nil {
			secondCompilationUnitContractCount += len(secondUnitSecondContractSource.Contracts)
		}
		assert.EqualValues(t, secondCompilationUnitContractCount, 2)
	})
}
