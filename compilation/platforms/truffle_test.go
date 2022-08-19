package platforms

import (
	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/utils/test_utils"
	"testing"
)

func TestTruffleCompilationAbsolutePath(t *testing.T) {
	// Copy our testdata over to our testing directory
	truffleDirectory := test_utils.CopyToTestDirectory(t, "testdata/truffle/basic_project/")

	// Create a solc provider
	truffleConfig := NewTruffleCompilationConfig(truffleDirectory)

	// Obtain our solc version and ensure we didn't encounter an error
	compilations, _, err := truffleConfig.Compile()
	assert.Nil(t, err)
	assert.True(t, len(compilations) > 0)
}
