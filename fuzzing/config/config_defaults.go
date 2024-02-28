package config

import (
	testChainConfig "github.com/crytic/medusa/chain/config"
	"github.com/crytic/medusa/compilation"
	"github.com/rs/zerolog"
	"math/big"
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
			Workers:                 10,
			WorkerResetLimit:        50,
			Timeout:                 0,
			TestLimit:               0,
			ShrinkLimit:             5_000,
			CallSequenceLength:      100,
			TargetContracts:         []string{},
			TargetContractsBalances: []*big.Int{},
			ConstructorArgs:         map[string]map[string]any{},
			CorpusDirectory:         "",
			CoverageEnabled:         true,
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
				StopOnFailedContractMatching: false,
				StopOnNoTests:                true,
				TestAllContracts:             false,
				TraceAll:                     false,
				AssertionTesting: AssertionTestingConfig{
					Enabled:         true,
					TestViewMethods: false,
					PanicCodeConfig: PanicCodeConfig{
						FailOnAssertion: true,
					},
				},
				PropertyTesting: PropertyTestingConfig{
					Enabled: true,
					TestPrefixes: []string{
						"property_",
					},
				},
				OptimizationTesting: OptimizationTestingConfig{
					Enabled: true,
					TestPrefixes: []string{
						"optimize_",
					},
				},
			},
			TestChainConfig: *chainConfig,
		},
		Compilation: compilationConfig,
		Logging: LoggingConfig{
			Level:        zerolog.InfoLevel,
			LogDirectory: "",
			NoColor:      false,
		},
	}

	// Return the project configuration
	return projectConfig, nil
}
