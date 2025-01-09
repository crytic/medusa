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
	}

	// Return the generated configuration.
	return config, nil
}
