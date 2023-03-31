package platforms

import (
	"github.com/crytic/medusa/utils/testutils"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

// TestSolcVersion ensures that a version of solc could be obtained and is installed
// on the system.
func TestSolcVersion(t *testing.T) {
	// Obtain our solc version and ensure we didn't encounter an error
	_, err := GetSystemSolcVersion()
	assert.NoError(t, err)
}

// TestSimpleSolcCompilationAbsolutePath tests that a single contract should be able to be compiled
// with an absolute target path in our platform config.
func TestSimpleSolcCompilationAbsolutePath(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := testutils.CopyToTestDirectory(t, "testdata/solc/SimpleContract.sol")

	// Execute our tests in the given test path
	testutils.ExecuteInDirectory(t, contractPath, func() {
		// Create a solc provider
		solc := NewSolcCompilationConfig(contractPath)

		// Obtain our compilations and ensure we didn't encounter an error
		compilations, _, err := solc.Compile()
		assert.NoError(t, err)
		assert.True(t, len(compilations) > 0)
	})
}

// TestSimpleSolcCompilationRelativePath tests that a single contract should be able to be compiled
// with a relative target path in our platform config.
func TestSimpleSolcCompilationRelativePath(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := testutils.CopyToTestDirectory(t, "testdata/solc/SimpleContract.sol")
	contractName := filepath.Base(contractPath)

	// Execute our tests in the given test path
	testutils.ExecuteInDirectory(t, contractPath, func() {
		// Create a solc provider
		solc := NewSolcCompilationConfig(contractName)

		// Obtain our solc version and ensure we didn't encounter an error
		compilations, _, err := solc.Compile()
		assert.NoError(t, err)
		assert.True(t, len(compilations) > 0)
	})
}

// TestFailedSolcCompilation tests that a single contract of invalid form should fail compilation.
func TestFailedSolcCompilation(t *testing.T) {
	// Copy our testdata over to our testing directory
	contractPath := testutils.CopyToTestDirectory(t, "testdata/solc/FailedCompilationContract.sol")

	// Execute our tests in the given test path
	testutils.ExecuteInDirectory(t, contractPath, func() {
		// Create a solc provider
		solc := NewSolcCompilationConfig(contractPath)

		// Obtain our solc version and ensure we didn't encounter an error
		compilations, _, err := solc.Compile()
		assert.NotNil(t, err)
		assert.True(t, len(compilations) == 0)
	})
}
