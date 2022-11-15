package fuzzing

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/compilation"
	"github.com/trailofbits/medusa/compilation/platforms"
	"github.com/trailofbits/medusa/fuzzing/config"
	"github.com/trailofbits/medusa/utils/testutils"
	"testing"
)

// fuzzerTest is an interface which is used to implement different test structures to invoke a test against the Fuzzer.
type fuzzerTest interface {
	// run describes a method which can be called abstractly, which takes data from an implementing structure and
	// runs a test against the Fuzzer with it.
	run(t *testing.T)
}

// runFuzzerTest takes a given fuzzerTest and testing state and executes the fuzzerTest with that state.
// It is a wrapper for test.Run(t).
func runFuzzerTest(t *testing.T, test fuzzerTest) {
	test.run(t)
}

// fuzzerSolcFileTest describes a test to be run against the Fuzzer using a single Solidity contract file.
type fuzzerSolcFileTest struct {
	// filePath describes the relative path from the current package to the source file to be tested.
	filePath string

	// configUpdates is a function which can be used to create updates to the default project configuration used for
	// testing, allowing for greater configuration of tests.
	configUpdates func(config *config.ProjectConfig)

	// method is the actual testing logic to execute once a Fuzzer has been created with the previously mentioned
	// project configuration, and the relevant testing context has been created.
	method func(fc *fuzzerTestContext)
}

// run implements fuzzerTest.run for a single Solidity test file described by the fuzzerSolcFileTest.
func (c *fuzzerSolcFileTest) run(t *testing.T) {
	// Print a status message
	fmt.Printf("##############################################################\n")
	fmt.Printf("Fuzzing '%s'...\n", c.filePath)
	fmt.Printf("##############################################################\n")

	// Copy our target file to our test directory
	contractTestPath := testutils.CopyToTestDirectory(t, c.filePath)

	// Run the test in our temporary test directory to avoid artifact pollution.
	testutils.ExecuteInDirectory(t, contractTestPath, func() {
		// Create a default solc platform config
		solcPlatformConfig := platforms.NewSolcCompilationConfig(contractTestPath)

		// Wrap the platform config in a compilation config
		compilationConfig, err := compilation.NewCompilationConfigFromPlatformConfig(solcPlatformConfig)
		assert.NoError(t, err)

		// Now create a project configuration.
		projectConfig := getFuzzerTestingProjectConfig(compilationConfig)

		// Run our config updates method provided for this test case.
		if c.configUpdates != nil {
			c.configUpdates(projectConfig)
		}

		// Run our test case
		executeFuzzerTestMethodInternal(t, projectConfig, c.method)
	})
}
