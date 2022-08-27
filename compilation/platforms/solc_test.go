package platforms

import (
	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/utils/test_utils"
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

	// Execute our tests in the given test path
	test_utils.ExecuteInDirectory(t, contractPath, func() {
		// Create a solc provider
		solc := NewSolcCompilationConfig(contractPath)

		// Obtain our solc version and ensure we didn't encounter an error
		compilations, _, err := solc.Compile()
		assert.Nil(t, err)
		assert.True(t, len(compilations) > 0)
	})
}

func TestSimpleSolcCompilationRelativePath(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := test_utils.CopyToTestDirectory(t, "testdata/solc/SimpleContract.sol")
	contractName := filepath.Base(contractPath)

	// Execute our tests in the given test path
	test_utils.ExecuteInDirectory(t, contractPath, func() {
		// Create a solc provider
		solc := NewSolcCompilationConfig(contractName)

		// Obtain our solc version and ensure we didn't encounter an error
		compilations, _, err := solc.Compile()
		assert.Nil(t, err)
		assert.True(t, len(compilations) > 0)
	})
}
