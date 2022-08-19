package platforms

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/utils/test_utils"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCryticSingleFileNoArgsAbsolutePath tests compilation of a single with no additional arguments and absolute path provided
func TestCryticSingleFileNoArgsAbsolutePath(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := test_utils.CopyToTestDirectory(t, "testdata/solc/SimpleContract.sol")

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
}

// TestCryticSingleFileNoArgsRelativePath tests compilation of a single with no additional arguments and relative path provided
func TestCryticSingleFileNoArgsRelativePath(t *testing.T) {
	// Backup our old working directory
	cwd, err := os.Getwd()
	assert.Nil(t, err)

	// Copy our testdata over to our testing directory
	contractPath := test_utils.CopyToTestDirectory(t, "testdata/solc/SimpleContract.sol")

	// Obtain the contract directory and cd to it.
	contractDirectory := filepath.Dir(contractPath)
	err = os.Chdir(contractDirectory)
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
	// Two contracts in SimpleContract.sol. Need to add /private for some weird symlink issue
	assert.True(t, len(compilations[0].Sources["/private"+contractPath].Contracts) == 2)

	// Restore our working directory (we must leave the test directory or else clean up will fail post testing)
	err = os.Chdir(cwd)
	assert.Nil(t, err)
}

// TestCryticSingleFileBadArgs tests compilation of a single with unaccepted or bad arguments
// (e.g. export-dir, export-format)
func TestCryticSingleFileBadArgs(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := test_utils.CopyToTestDirectory(t, "testdata/solc/SimpleContract.sol")

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
}

// TestCryticDirectoryNoArgs tests compilation of a whole directory with no addition arguments provided
func TestCryticDirectoryNoArgs(t *testing.T) {
	// Backup our old working directory
	cwd, err := os.Getwd()
	assert.Nil(t, err)

	// Copy our testdata over to our testing directory
	contractDirectory := test_utils.CopyToTestDirectory(t, "testdata/hardhat/basic_project/")
	fmt.Printf("contract directory: %v\n", contractDirectory)
	// Change wd and run npm install
	err = os.Chdir(contractDirectory)
	assert.Nil(t, err)
	err = exec.Command("npm", "install").Run()
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
	// Compilation unit ordering is non-deterministic in JSON output
	// All we care about is that each comp unit has two contracts for one or the other file
	// TODO: might be a better way to do this?
	firstCompilationUnitHasTwoContracts :=
		(len(compilations[0].Sources["/private"+contractDirectory+"/contracts/SecondContract.sol"].Contracts) == 2 &&
			len(compilations[0].Sources["/private"+contractDirectory+"/contracts/FirstContract.sol"].Contracts) == 0) ||
			(len(compilations[0].Sources["/private"+contractDirectory+"/contracts/FirstContract.sol"].Contracts) == 2 &&
				len(compilations[0].Sources["/private"+contractDirectory+"/contracts/SecondContract.sol"].Contracts) == 0)
	secondCompilationUnitHasTwoContracts :=
		(len(compilations[1].Sources["/private"+contractDirectory+"/contracts/SecondContract.sol"].Contracts) == 2 &&
			len(compilations[1].Sources["/private"+contractDirectory+"/contracts/FirstContract.sol"].Contracts) == 0) ||
			(len(compilations[1].Sources["/private"+contractDirectory+"/contracts/FirstContract.sol"].Contracts) == 2 &&
				len(compilations[1].Sources["/private"+contractDirectory+"/contracts/SecondContract.sol"].Contracts) == 0)

	assert.True(t, firstCompilationUnitHasTwoContracts)
	assert.True(t, secondCompilationUnitHasTwoContracts)

	// Restore our working directory (we must leave the test directory or else clean up will fail post testing)
	err = os.Chdir(cwd)
	assert.Nil(t, err)
}
