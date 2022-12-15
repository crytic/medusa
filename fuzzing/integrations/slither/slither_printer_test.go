package slither

import (
	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/compilation/platforms"
	"github.com/trailofbits/medusa/fuzzing/valuegeneration"
	"github.com/trailofbits/medusa/utils/testutils"
	"testing"
)

// TestRunPrinterAndValueSetAddition will test that slither's echidna printer runs successfully and then will also make
// sure that all the constants can be added to the base value set.
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
		assert.Equal(t, len(slitherData.GetConstantsInContract("SlitherConstants")), 4)

		// Now we will add these constants to the base value set
		vs := valuegeneration.NewValueSet()
		err = slitherData.AddConstantsToValueSet(vs)
		assert.NoError(t, err)

		// Check to make sure that the three integer values are in the value set
		for _, val := range vs.Integers() {
			if val.Uint64() != 2 && val.Uint64() != 0 && val.Uint64() != 2e18 {
				assert.Fail(t, "constants are not set in the base value set")
			}
		}
	})
}
