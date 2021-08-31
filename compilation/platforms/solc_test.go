package platforms

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSolcVersion(t *testing.T) {
	// Obtain our solc version and ensure we didn't encounter an error
	_, err := GetSystemSolcVersion()
	assert.Nil(t, err)
}

func TestSolcCompile(t *testing.T) {
	// Create a solc provider
	solc := NewSolcCompilationConfig("../testContracts/test.sol")

	// Obtain our solc version and ensure we didn't encounter an error
	compilations, err := solc.Compile()
	assert.Nil(t, err)
	assert.True(t, len(compilations) > 0)
}
