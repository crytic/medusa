package platforms

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path"
	"testing"
)

func TestSimpleCryticCompilation(t *testing.T) {
	// Define our contract source code
	contractSource := `
contract SimpleCryticCompilation {
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
	tempDir := t.TempDir()
	contractPath := path.Join(tempDir, "simple_crytic_compilation.sol")
	err := ioutil.WriteFile(contractPath, []byte(contractSource), 0777)
	assert.Nil(t, err)

	// Create a solc provider
	fmt.Printf("directory is %s\n", tempDir)
	cryticConfig := NewCryticCompilationConfig(contractPath)

	// Obtain our solc version and ensure we didn't encounter an error
	compilations, _, err := cryticConfig.Compile()
	assert.Nil(t, err)
	assert.True(t, len(compilations) > 0)
}
