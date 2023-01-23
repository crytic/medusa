package config

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"
	"github.com/trailofbits/medusa/chain"
	"github.com/trailofbits/medusa/compilation"
)

// GetDefaultProjectConfig obtains a default configuration for a project. It populates a default compilation config
// based on the provided platform, or a nil one if an empty string is provided.
func GetDefaultProjectConfig(platform string) (*ProjectConfig, error) {
	var (
		compilationConfig *compilation.CompilationConfig
		err               error
	)
	if platform != "" {
		compilationConfig, err = compilation.NewCompilationConfig(platform)
		if err != nil {
			return nil, err
		}
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
				"0x1111111111111111111111111111111111111111",
				"0x2222222222222222222222222222222222222222",
				"0x3333333333333333333333333333333333333333",
			},
			DeployerAddress:        "0x1111111111111111111111111111111111111111",
			MaxBlockNumberDelay:    60480,
			MaxBlockTimestampDelay: 604800,
			BlockGasLimit:          125_000_000,
			TransactionGasLimit:    12_500_000,
			Testing: TestingConfig{
				StopOnFailedTest:             true,
				StopOnFailedContractMatching: true,
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
		},
		Compilation: compilationConfig,
		ChainConfig: &chain.TestChainConfig{
			CoreConfig: &params.ChainConfig{
				ChainID:                       big.NewInt(1),
				HomesteadBlock:                big.NewInt(0),
				DAOForkBlock:                  nil,
				DAOForkSupport:                false,
				EIP150Block:                   big.NewInt(0),
				EIP155Block:                   big.NewInt(0),
				EIP158Block:                   big.NewInt(0),
				ByzantiumBlock:                big.NewInt(0),
				ConstantinopleBlock:           big.NewInt(0),
				PetersburgBlock:               big.NewInt(0),
				IstanbulBlock:                 big.NewInt(0),
				MuirGlacierBlock:              big.NewInt(0),
				BerlinBlock:                   big.NewInt(0),
				LondonBlock:                   big.NewInt(0),
				ArrowGlacierBlock:             big.NewInt(0),
				GrayGlacierBlock:              big.NewInt(0),
				MergeNetsplitBlock:            nil,
				ShanghaiBlock:                 nil,
				CancunBlock:                   nil,
				TerminalTotalDifficulty:       nil,
				TerminalTotalDifficultyPassed: false,
				Ethash:                        new(params.EthashConfig),
				Clique:                        nil,
			},
		},
	}

	// Return the project configuration
	return projectConfig, nil
}
