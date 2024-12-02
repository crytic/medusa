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
			ForkModeEnabled: true,
			RpcUrl:          "http://localhost:8080/v2/ju1AhO8KhGNGQ4wOSRjTBaVFOY9_cORj",
			RpcBlock:        21310796,
			PoolSize:        5,
		},
	}

	// Return the generated configuration.
	return config, nil
}
