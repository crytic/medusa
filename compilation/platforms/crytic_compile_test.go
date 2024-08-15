package platforms

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crytic/medusa/compilation/types"
	"github.com/crytic/medusa/utils"
	"github.com/crytic/medusa/utils/testutils"
	"github.com/stretchr/testify/assert"
)

// testCryticGetCompiledSourceByBaseName checks if a given source file exists in a given compilation's map of sources.
// The source file is the file name of a specific file. This function simply checks one of the paths ends with
// this name. Avoid including any directories in case the path separators differ per system.
// Returns the types.CompiledSource (mapping value) associated to the path if it is found. Returns nil otherwise.
func testCryticGetCompiledSourceByBaseName(sources map[string]types.SourceArtifact, name string) *types.SourceArtifact {
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

// TestCryticSingleFileAbsolutePath tests compilation of a single smart contract using an absolute
// file path.
func TestCryticSingleFileAbsolutePath(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := testutils.CopyToTestDirectory(t, "testdata/solc/basic/SimpleContract.sol")

	// Execute our tests in the given test path
	testutils.ExecuteInDirectory(t, contractPath, func() {
		// Create our platform configuration
		config := NewCryticCompilationConfig(contractPath)

		// Compile the file
		compilations, _, err := config.Compile()
		// No failures
		assert.NoError(t, err)
		// One compilation object
		assert.EqualValues(t, 1, len(compilations))
		// One source because we specified one file
		assert.EqualValues(t, 1, len(compilations[0].SourcePathToArtifact))
		// Two contracts in SimpleContract.sol
		contractCount := 0
		for _, source := range compilations[0].SourcePathToArtifact {
			contractCount += len(source.Contracts)
		}
		assert.EqualValues(t, 2, contractCount)
	})
}

// TestCryticSingleFileRelativePathSameDirectory tests compilation of a single smart contract using a relative
// file path in the working directory.
func TestCryticSingleFileRelativePathSameDirectory(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := testutils.CopyToTestDirectory(t, "testdata/solc/basic/SimpleContract.sol")
	contractName := filepath.Base(contractPath)

	// Execute our tests in the given test path
	testutils.ExecuteInDirectory(t, contractPath, func() {
		// Create our platform configuration
		config := NewCryticCompilationConfig(contractName)

		// Compile the file
		compilations, _, err := config.Compile()
		// No failures
		assert.NoError(t, err)
		// One compilation object
		assert.EqualValues(t, 1, len(compilations))
		// One source because we specified one file
		assert.EqualValues(t, 1, len(compilations[0].SourcePathToArtifact))
		// Two contracts in SimpleContract.sol
		contractCount := 0
		for _, source := range compilations[0].SourcePathToArtifact {
			contractCount += len(source.Contracts)
		}
		assert.EqualValues(t, 2, contractCount)
	})
}

// TestCryticSingleFileRelativePathChildDirectory tests compilation of a single smart contract using a relative
// file path in a child directory of the working directory.
func TestCryticSingleFileRelativePathChildDirectory(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := testutils.CopyToTestDirectory(t, "testdata/solc/basic/SimpleContract.sol")

	// Move it to a subdirectory
	contractDirectory := filepath.Dir(contractPath)
	relativeRelocatedPath := filepath.Join("child_dir", "SimpleContract.sol")
	absoluteRelocatedPath := filepath.Join(contractDirectory, relativeRelocatedPath)
	err := utils.CopyFile(contractPath, absoluteRelocatedPath)
	assert.NoError(t, err)

	// Execute our tests in the given test path
	testutils.ExecuteInDirectory(t, contractDirectory, func() {
		// Create our platform configuration
		config := NewCryticCompilationConfig(relativeRelocatedPath)
		config.ExportDirectory = "custom_export_directory"

		// Compile the file
		compilations, _, err := config.Compile()
		// No failures
		assert.NoError(t, err)
		// One compilation object
		assert.EqualValues(t, 1, len(compilations))
		// One source because we specified one file
		assert.EqualValues(t, 1, len(compilations[0].SourcePathToArtifact))
		// Two contracts in SimpleContract.sol
		contractCount := 0
		for _, source := range compilations[0].SourcePathToArtifact {
			contractCount += len(source.Contracts)
		}
		assert.EqualValues(t, 2, contractCount)

		// Verify our build directory exists
		dirInfo, err := os.Stat(config.ExportDirectory)
		assert.NoError(t, err)
		assert.True(t, dirInfo.IsDir())
	})
}

// TestCryticSingleFileBuildDirectoryArgRelativePath tests compilation of a single contract with the buildDirectory arg and
// a relative path provided.
func TestCryticSingleFileBuildDirectoryArgRelativePath(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := testutils.CopyToTestDirectory(t, "testdata/solc/basic/SimpleContract.sol")

	// Execute our tests in the given test path
	testutils.ExecuteInDirectory(t, contractPath, func() {
		// Obtain the contract directory and cd to it.
		contractDirectory := filepath.Dir(contractPath)
		err := os.Chdir(contractDirectory)
		assert.NoError(t, err)

		// Obtain the filename
		contractName := filepath.Base(contractPath)

		// Create our platform configuration
		config := NewCryticCompilationConfig(contractName)
		// Custom export directory
		config.ExportDirectory = "export_directory"
		// Compile the file
		compilations, _, err := config.Compile()
		// No failures
		assert.NoError(t, err)
		// One compilation object
		assert.EqualValues(t, 1, len(compilations))
		// One source because we specified one file
		assert.EqualValues(t, 1, len(compilations[0].SourcePathToArtifact))
		// Two contracts in SimpleContract.sol.
		compiledSource := testCryticGetCompiledSourceByBaseName(compilations[0].SourcePathToArtifact, contractName)
		assert.NotNil(t, compiledSource, "source file could not be resolved in compilation sources")
		assert.EqualValues(t, 2, len(compiledSource.Contracts))
	})
}

// TestCryticSingleFileBadArgs tests compilation of a single contract with unaccepted or bad arguments
// (e.g. export-dir, export-format)
func TestCryticSingleFileBadArgs(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := testutils.CopyToTestDirectory(t, "testdata/solc/basic/SimpleContract.sol")

	// Execute our tests in the given test path
	testutils.ExecuteInDirectory(t, contractPath, func() {
		// Create a config and verify it compiles without any bad arguments first.
		config := NewCryticCompilationConfig(contractPath)
		_, _, err := config.Compile()
		assert.NoError(t, err)

		// Define arguments to validate are not allowed on a config
		failingArgs := []string{
			"--export-format",
			"--export-dir",
			"--bad-arg", // arbitrary unknown argument
		}

		// Validate the use of each argument results in an error when compiling
		for _, failingArg := range failingArgs {
			config = NewCryticCompilationConfig(contractPath)
			config.Args = append(config.Args, failingArg)
			_, _, err := config.Compile()
			assert.Error(t, err)
		}
	})
}

// TestCryticMultipleFiles tests compilation of a single target that inherits from another file.
func TestCryticMultipleFiles(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := testutils.CopyToTestDirectory(t, "testdata/solc/basic/")

	// Execute our tests in the given test path
	testutils.ExecuteInDirectory(t, contractPath, func() {
		// Create our platform configuration
		config := NewCryticCompilationConfig("DerivedContract.sol")

		// Compile the file
		compilations, _, err := config.Compile()
		assert.NoError(t, err)

		// Verify there is one compilation object
		assert.EqualValues(t, 1, len(compilations))
		// Verify there are two sources
		assert.EqualValues(t, 2, len(compilations[0].SourcePathToArtifact))

		// Verify there are three contracts
		contractCount := 0
		for _, source := range compilations[0].SourcePathToArtifact {
			contractCount += len(source.Contracts)
		}
		assert.EqualValues(t, 3, contractCount)
	})
}

// TestCryticDirectoryNoArgs tests compilation of a hardhat directory with no addition arguments provided
func TestCryticDirectoryNoArgs(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractDirectory := testutils.CopyToTestDirectory(t, "testdata/hardhat/basic_project/")

	// Execute our tests in the given test path
	testutils.ExecuteInDirectory(t, contractDirectory, func() {
		// Run npm install
		err := exec.Command("npm", "install").Run()
		assert.NoError(t, err)

		// Create our platform configuration and compile the project
		config := NewCryticCompilationConfig(contractDirectory)
		compilations, _, err := config.Compile()

		// No failures
		assert.NoError(t, err)

		// Two compilation objects
		assert.EqualValues(t, 2, len(compilations))
		// One source per compilation unit
		assert.EqualValues(t, 1, len(compilations[0].SourcePathToArtifact))
		assert.EqualValues(t, 1, len(compilations[1].SourcePathToArtifact))

		// Obtain the compiled source from both compilation units
		firstContractName := "FirstContract.sol"
		secondContractName := "SecondContract.sol"
		firstUnitFirstContractSource := testCryticGetCompiledSourceByBaseName(compilations[0].SourcePathToArtifact, firstContractName)
		firstUnitSecondContractSource := testCryticGetCompiledSourceByBaseName(compilations[0].SourcePathToArtifact, secondContractName)
		secondUnitFirstContractSource := testCryticGetCompiledSourceByBaseName(compilations[1].SourcePathToArtifact, firstContractName)
		secondUnitSecondContractSource := testCryticGetCompiledSourceByBaseName(compilations[1].SourcePathToArtifact, secondContractName)

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
