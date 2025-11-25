package config

import (
	testChainConfig "github.com/crytic/medusa/chain/config"
	"github.com/crytic/medusa/compilation"
	"github.com/crytic/medusa/compilation/types"
	"github.com/rs/zerolog"
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

	// Obtain a default slither configuration
	slitherConfig, err := types.NewDefaultSlitherConfig()
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
			PruneFrequency:          5,
			TargetContracts:         []string{},
			TargetContractsBalances: []*ContractBalance{},
			PredeployedContracts:    map[string]string{},
			ConstructorArgs:         map[string]map[string]any{},
			CorpusDirectory:         "",
			CoverageEnabled:         true,
			CoverageFormats:         []string{"html", "lcov"},
			CoverageExclusions:      []string{},
			SenderAddresses: []string{
				"0x10000",
				"0x20000",
				"0x30000",
			},
			DeployerAddress:        "0x30000",
			MaxBlockNumberDelay:    60480,
			MaxBlockTimestampDelay: 604800,
			TransactionGasLimit:    12_500_000,
			RevertReporterEnabled:  false,
			Testing: TestingConfig{
				StopOnFailedTest:             true,
				StopOnFailedContractMatching: false,
				StopOnNoTests:                true,
				TestViewMethods:              true,
				TestAllContracts:             false,
				Verbosity:                    1,
				TargetFunctionSignatures:     []string{},
				ExcludeFunctionSignatures:    []string{},
				AssertionTesting: AssertionTestingConfig{
					Enabled: true,
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
		Slither:     slitherConfig,
		Logging: LoggingConfig{
			Level:        zerolog.InfoLevel,
			LogDirectory: "",
			NoColor:      false,
			EnableTUI:    false, // Disabled by default for backwards compatibility
		},
	}

	// Return the project configuration
	return projectConfig, nil
}
