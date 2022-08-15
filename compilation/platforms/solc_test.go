package platforms

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path"
	"testing"
)

func TestSolcVersion(t *testing.T) {
	// Obtain our solc version and ensure we didn't encounter an error
	_, err := GetSystemSolcVersion()
	assert.Nil(t, err)
}

func TestSimpleSolcCompilation(t *testing.T) {
	// Define our contract source code
	contractSource := `
contract SimpleSolcCompilation {
    uint x1;
    uint x2;

    function setx1(uint val) public {
        x1 = val;
    }

    function setx2(uint val) public {
        x2 = val;
    }

    function medusa_set_x1_x2_sequence() public view returns (bool) {
        return x1 != x2 * 3 || x1 == 0;
    }
}`

	// Write the contract out to our temporary test directory
	contractPath := path.Join(t.TempDir(), "simple_solc_compilation.sol")
	err := ioutil.WriteFile(contractPath, []byte(contractSource), 0644)
	assert.Nil(t, err)

	// Create a solc provider
	solc := NewSolcCompilationConfig(contractPath)

	// Obtain our solc version and ensure we didn't encounter an error
	compilations, _, err := solc.Compile()
	assert.Nil(t, err)
	assert.True(t, len(compilations) > 0)
}
