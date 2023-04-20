package platforms

import (
	"github.com/crytic/medusa/utils/testutils"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestTruffleCompilationAbsolutePath tests compilation of a truffle project with an absolute project path.
func TestTruffleCompilationAbsolutePath(t *testing.T) {
	// Copy our testdata over to our testing directory
	truffleDirectory := testutils.CopyToTestDirectory(t, "testdata/truffle/basic_project/")

	// Execute our tests in the given test path
	testutils.ExecuteInDirectory(t, truffleDirectory, func() {
		// Create a solc provider
		truffleConfig := NewTruffleCompilationConfig(truffleDirectory)

		// Obtain our solc version and ensure we didn't encounter an error
		compilations, _, err := truffleConfig.Compile()
		assert.NoError(t, err)
		assert.True(t, len(compilations) > 0)
	})
}
