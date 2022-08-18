package platforms

import (
	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/utils/test_utils"
	"os"
	"path/filepath"
	"testing"
)

func TestSolcVersion(t *testing.T) {
	// Obtain our solc version and ensure we didn't encounter an error
	_, err := GetSystemSolcVersion()
	assert.Nil(t, err)
}

func TestSimpleSolcCompilationAbsolutePath(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := test_utils.CopyToTestDirectory(t, "testdata/solc/SimpleContract.sol")

	// Create a solc provider
	solc := NewSolcCompilationConfig(contractPath)

	// Obtain our solc version and ensure we didn't encounter an error
	compilations, _, err := solc.Compile()
	assert.Nil(t, err)
	assert.True(t, len(compilations) > 0)
}

func TestSimpleSolcCompilationRelativePath(t *testing.T) {
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

	// Create a solc provider
	solc := NewSolcCompilationConfig(contractName)

	// Obtain our solc version and ensure we didn't encounter an error
	compilations, _, err := solc.Compile()
	assert.Nil(t, err)
	assert.True(t, len(compilations) > 0)

	// Restore our working directory (we must leave the test directory or else clean up will fail post testing)
	err = os.Chdir(cwd)
	assert.Nil(t, err)
}
