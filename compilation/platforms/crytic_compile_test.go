package platforms

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path"
	"testing"
)

// TODO: This whole file will change after the dev/test PR is merged

// SetupContracts is a helper function that will perform different actions based on the values of isFile and
// twoContracts. If isFile is true, the function will return the path of the contract file. If isFile is false,
// the function will return the path of the directory. If twoContracts is false, only 1 contract is written to the
// working directory and if twoContracts is true, two contracts are written to the working directory.
func SetupContracts(t *testing.T, isFile bool, twoContracts bool) string {
	contractOne := `
	contract ContractOne {
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
	contractTwo := `
	abstract contract AbstractContractTwo {
		uint x;		
		function setx(uint val) public {
			x = val;
		}
	}
	contract ContractTwo {
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
	contractPathOne := path.Join(tempDir, "crytic_one.sol")
	err := ioutil.WriteFile(contractPathOne, []byte(contractOne), 0644)
	assert.Nil(t, err)
	if twoContracts {
		contractPathTwo := path.Join(tempDir, "crytic_two.sol")
		err := ioutil.WriteFile(contractPathTwo, []byte(contractTwo), 0644)
		assert.Nil(t, err)
	}
	if isFile {
		return contractPathOne
	}
	return tempDir
}

// TestCryticSingleFileNoArgs tests compilation of a single with no additional arguments provided
func TestCryticSingleFileNoArgs(t *testing.T) {
	// Setup contract
	isFile := true
	twoContracts := false
	filePath := SetupContracts(t, isFile, twoContracts)

	// Create a crytic-compile provider
	cryticConfig := NewCryticCompilationConfig(filePath)

	compilations, _, err := cryticConfig.Compile()
	// No failures
	assert.Nil(t, err)
	assert.True(t, len(compilations) == 1)                                // One compilation object
	assert.True(t, len(compilations[0].Sources) == 1)                     // One source because we specified one file
	assert.True(t, len(compilations[0].Sources[filePath].Contracts) == 1) // One contract in crytic.sol
}

// TestCryticSingleFileBadArgs tests compilation of a single with unaccepted or bad arguments
// (e.g. export-dir, export-format)
func TestCryticSingleFileBadArgs(t *testing.T) {
	// Setup contract
	isFile := true
	twoContracts := false
	filePath := SetupContracts(t, isFile, twoContracts)

	// Create a crytic-compile provider
	cryticConfig := NewCryticCompilationConfig(filePath)
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
	// Setup contracts
	isFile := false
	twoContracts := true
	filePath := SetupContracts(t, isFile, twoContracts)

	// Create a crytic-compile provider
	cryticConfig := NewCryticCompilationConfig(filePath)
	compilations, _, err := cryticConfig.Compile()
	// No failures
	assert.Nil(t, err)
	assert.True(t, len(compilations) == 2)                                                             // One compilation object
	assert.True(t, len(compilations[0].Sources) == 1)                                                  // Two sources because we specified two files
	assert.True(t, len(compilations[1].Sources) == 1)                                                  // Two sources because we specified two files
	assert.True(t, len(compilations[0].Sources["/private"+filePath+"/crytic_one.sol"].Contracts) == 1) // One contract in crytic_one.sol
	assert.True(t, len(compilations[1].Sources["/private"+filePath+"/crytic_two.sol"].Contracts) == 2) // Two contracts in crytic_two.sol
}
