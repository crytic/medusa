package config

// DefaultTestChainConfig obtains a default configuration for a chain.TestChain.
// Returns a TestChainConfig populated with default values.
func DefaultTestChainConfig() (*TestChainConfig, error) {
	// Create a default config and return it.
	config := &TestChainConfig{
		CodeSizeCheckDisabled: true,
		CheatCodeConfig: CheatCodeConfig{
			CheatCodesEnabled: true,
			EnableFFI:         false,
		},
		SkipAccountChecks: true,
		ForkConfig: ForkConfig{
			ForkModeEnabled: false,
			RpcUrl:          "",
			RpcBlock:        1,
			PoolSize:        20,
		},
	}

	// Return the generated configuration.
	return config, nil
}
