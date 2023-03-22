package config

import (
	testChainConfig "github.com/trailofbits/medusa/chain/config"
	"github.com/trailofbits/medusa/compilation"
)

// GetDefaultProjectConfig obtains a default configuration for a project. It populates a default compilation config
// based on the provided platform, or a nil one if an empty string is provided.
func GetDefaultProjectConfig(platform string) (*ProjectConfig, error) {
	var (
		compilationConfig *compilation.CompilationConfig
		chainConfig       *testChainConfig.TestChainConfig
		err               error
	)

	// Try to obtain a default compilation config for this platform.
	if platform != "" {
		compilationConfig, err = compilation.NewCompilationConfig(platform)
		if err != nil {
			return nil, err
		}
	}

	// Try to obtain a default chain config.
	chainConfig, err = testChainConfig.DefaultTestChainConfig()
	if err != nil {
		return nil, err
	}

	// Create a project configuration
	projectConfig := &ProjectConfig{
		Fuzzing: FuzzingConfig{
			Workers:            10,
			WorkerResetLimit:   50,
			Timeout:            0,
			TestLimit:          0,
			CallSequenceLength: 100,
			DeploymentOrder:    []string{},
			ConstructorArgs:    map[string]map[string]any{},
			CorpusDirectory:    "",
			CoverageEnabled:    true,
			SenderAddresses: []string{
				"0x10000",
				"0x20000",
				"0x30000",
			},
			DeployerAddress:        "0x30000",
			MaxBlockNumberDelay:    60480,
			MaxBlockTimestampDelay: 604800,
			BlockGasLimit:          125_000_000,
			TransactionGasLimit:    12_500_000,
			Testing: TestingConfig{
				StopOnFailedTest:             true,
				StopOnFailedContractMatching: true,
				TestAllContracts:             false,
				TraceAll:                     false,
				AssertionTesting: AssertionTestingConfig{
					Enabled:         false,
					TestViewMethods: false,
				},
				PropertyTesting: PropertyTestConfig{
					Enabled: true,
					TestPrefixes: []string{
						"fuzz_",
					},
				},
			},
			TestChainConfig: *chainConfig,
		},
		Compilation: compilationConfig,
	}

	// Return the project configuration
	return projectConfig, nil
}
