package fuzzing

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/trailofbits/medusa/compilation"
	"github.com/trailofbits/medusa/compilation/platforms"
	"github.com/trailofbits/medusa/events"
	"github.com/trailofbits/medusa/fuzzing/config"
	"github.com/trailofbits/medusa/utils/testutils"
	"os/exec"
	"strconv"
	"testing"
)

// fuzzerTestingContext holds the current fuzzing testing context and can be used for post-execution checks
// such as testing that events were properly emitted
type fuzzerTestingContext struct {
	fuzzer              *Fuzzer
	postExecutionChecks []func(*testing.T, *fuzzerTestingContext)
	eventCounter        map[string]int
}

// newFuzzerTestContext creates a new fuzzerTestingContext
func newFuzzerTestContext(fuzzer *Fuzzer) *fuzzerTestingContext {
	return &fuzzerTestingContext{
		fuzzer:              fuzzer,
		postExecutionChecks: make([]func(*testing.T, *fuzzerTestingContext), 0),
		eventCounter:        make(map[string]int),
	}
}

// assertFailedTestExists will check to see whether there are any failed tests. If `expectFailure` is false, then there should
// be no failed tests
func assertFailedTestExists(t *testing.T, fuzzer *Fuzzer, expectFailure bool) {
	// Ensure we captured a failed test, if expected
	if expectFailure {
		assert.True(t, len(fuzzer.TestCasesWithStatus(TestCaseStatusFailed)) > 0, "Fuzz test could not be solved before timeout ("+strconv.Itoa(fuzzer.config.Fuzzing.Timeout)+" seconds)")
	} else {
		assert.True(t, len(fuzzer.TestCasesWithStatus(TestCaseStatusFailed)) == 0, "Fuzz test found a violated property test when it should not have")
	}
}

// assertCoverageCollected will check to see whether we captured some corpus items. If `expectCoverage` is false or
// coverage is not enabled, then there should be no coverage
func assertCoverageCollected(t *testing.T, fuzzer *Fuzzer, expectCoverage bool) {
	// Ensure we captured some coverage
	if expectCoverage {
		assert.True(t, fuzzer.corpus.CallSequenceCount() > 0, "No coverage was captured")
	}
	// If we don't expect coverage, or it is not enabled, we should not get any coverage
	if !expectCoverage || !fuzzer.config.Fuzzing.CoverageEnabled {
		assert.True(t, fuzzer.corpus.CallSequenceCount() == 0, "Coverage was captured")
	}
}

// setupFuzzerEnvironment will create the fuzzer object in a test directory and pass control flow to an anonymous
// `method`. This `method` can then run the fuzzer and test whatever it needs to. After the `method` returns, a set of
// post-execution checks are performed. The `method` can determine what these checks are.
func setupFuzzerEnvironment(t *testing.T, projectConfig *config.ProjectConfig, target string, method func(fctx *fuzzerTestingContext), commands ...[]string) {
	// Print a status message
	fmt.Printf("##############################################################\n")
	fmt.Printf("Fuzzing '%s'...\n", target)
	fmt.Printf("##############################################################\n")
	// Copy our target file to our test directory
	testContractPath := testutils.CopyToTestDirectory(t, target)
	testutils.ExecuteInDirectory(t, testContractPath, func() {
		// Run arbitrary commands that might be needed to set up the environment (e.g. npm install)
		// Note that each command must be in the form [name, args...]
		for _, command := range commands {
			err := exec.Command(command[0], command[1:]...).Run()
			assert.NoError(t, err)
		}

		// Create a fuzzer instance
		fuzzer, err := NewFuzzer(*projectConfig)
		assert.NoError(t, err)

		// Create new fuzzing context
		fctx := newFuzzerTestContext(fuzzer)
		// Run anonymous function with a pointer to the current fuzzer instance
		method(fctx)

		// Call post-execution checks
		for _, fxn := range fctx.postExecutionChecks {
			fxn(t, fctx)
		}
	})
}

// getDefaultFuzzingConfig obtains the default configuration for tests.
func getDefaultFuzzingConfig() *config.FuzzingConfig {
	return &config.FuzzingConfig{
		Workers:                  10,
		WorkerDatabaseEntryLimit: 10000,
		Timeout:                  30,
		TestLimit:                0,
		MaxTxSequenceLength:      100,
		SenderAddresses: []string{
			"0x1111111111111111111111111111111111111111",
			"0x2222222222222222222222222222222222222222",
			"0x3333333333333333333333333333333333333333",
		},
		DeployerAddress: "0x1111111111111111111111111111111111111111",
		Testing: config.TestingConfig{
			StopOnFailedTest: true,
			AssertionTesting: config.AssertionTestingConfig{
				Enabled:         false,
				TestViewMethods: false,
			},
			PropertyTesting: config.PropertyTestConfig{
				Enabled: true,
				TestPrefixes: []string{
					"fuzz_",
				},
			},
		},
		CoverageEnabled: true,
		CorpusDirectory: "corpus",
	}
}

// getSolcCompilationConfig returns a compilationConfig object where solc will be the underlying compilation platform
func getSolcCompilationConfig(target string) (*compilation.CompilationConfig, error) {
	// Create a default solc platform config
	solcPlatformConfig := platforms.NewSolcCompilationConfig(target)

	// Wrap the platform config in a compilation config
	compilationConfig, err := compilation.NewCompilationConfigFromPlatformConfig(solcPlatformConfig)
	if err != nil {
		return nil, err
	}
	return compilationConfig, nil
}

// getCryticCompileCompilationConfig returns a compilationConfig object where crytic-compile will be the underlying compilation platform
func getCryticCompileCompilationConfig(target string) (*compilation.CompilationConfig, error) {
	// Create a default crytic-compile platform config
	cryticCompilationConfig := platforms.NewCryticCompilationConfig(target)

	// Wrap the platform config in a compilation config
	compilationConfig, err := compilation.NewCompilationConfigFromPlatformConfig(cryticCompilationConfig)
	if err != nil {
		return nil, err
	}
	return compilationConfig, nil
}

// getDefaultProjectConfig will create a default projectConfig where the FuzzingConfig is fixed while the compilationConfig
// is not.
func getDefaultProjectConfig(compilationConfig *compilation.CompilationConfig) *config.ProjectConfig {
	fuzzingConfig := getDefaultFuzzingConfig()
	projectConfig := &config.ProjectConfig{
		Fuzzing:     *fuzzingConfig,
		Compilation: compilationConfig,
	}
	return projectConfig
}

// getProjectConfig will create a projectConfig given a specific fuzzing and compilation config
func getProjectConfig(fuzzingConfig *config.FuzzingConfig, compilationConfig *compilation.CompilationConfig) *config.ProjectConfig {
	projectConfig := &config.ProjectConfig{
		Fuzzing:     *fuzzingConfig,
		Compilation: compilationConfig,
	}
	return projectConfig
}

// stopFuzzerOnFuzzerStartingEvent will stop the fuzzer after the OnFuzzerStarting event is emitted
func stopFuzzerOnFuzzerStartingEvent(event OnFuzzerStarting) {
	// Simply stop the fuzzer
	event.Fuzzer.Stop()
}

// expectEmittedEvent will subscribe to some event T, update the eventCounter for that event (when the event callback is
// triggered) and then also add a post execution check to make sure that the event was captured properly.
func expectEmittedEvent[T any](t *testing.T, fctx *fuzzerTestingContext, eventEmitter *events.EventEmitter[T]) {
	// Get the stringified event type for the mapping
	eventType := eventEmitter.EventType().String()
	// Subscribe to the event T and update the counter when the event is published
	eventEmitter.Subscribe(func(event T) {
		fctx.eventCounter[eventType] += 1
	})
	// Add a check to make sure that event T was published at least once
	fctx.postExecutionChecks = append(fctx.postExecutionChecks, func(t *testing.T, fctx *fuzzerTestingContext) {
		// TODO: Is this a bit too loose of a check? Should we allow the user to specify the post-condition here as a
		//  function arg?
		assertEventIsEmittedAtLeastOnce(t, fctx, eventType)
	})
}

// assertEventIsEmittedAtLeastOnce will check to make sure that a given eventType has been emitted at least once.
func assertEventIsEmittedAtLeastOnce(t *testing.T, fctx *fuzzerTestingContext, eventType string) {
	assert.Greater(t, fctx.eventCounter[eventType], 0, "Event was not emitted at all")
}
