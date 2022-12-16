package slither

import (
	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/compilation/platforms"
	"github.com/trailofbits/medusa/utils/testutils"
	"testing"
)

// TestRunPrinterAndValueSetAddition will test that slither's echidna printer runs successfully and make sure all the
// constants show up in the SlitherData.Constants field.
func TestRunPrinterAndValueSetAddition(t *testing.T) {
	// Set target and copy it to the test directory
	target := "testdata/contracts/slither_constants.sol"
	contractTestPath := testutils.CopyToTestDirectory(t, target)
	testutils.ExecuteInDirectory(t, contractTestPath, func() {
		// Create compilation object and compile the target
		solcConfig := platforms.NewSolcCompilationConfig(contractTestPath)
		_, _, err := solcConfig.Compile()
		assert.NoError(t, err)

		// Run the slither printer
		slitherData, err := RunPrinter(contractTestPath)
		assert.NoError(t, err)

		// Ensure that there are four constants from the SlitherConstants contract
		constants := slitherData.GetConstantsInContract("SlitherConstants")
		assert.Equal(t, len(constants), 4)

		// Ensure that there are 2 constants for each function
		constants = slitherData.GetConstantsInMethod("SlitherConstants", "echidna_uint()")
		assert.Equal(t, len(constants), 2)

		constants = slitherData.GetConstantsInMethod("SlitherConstants", "echidna_ether()")
		assert.Equal(t, len(constants), 2)
	})
}
