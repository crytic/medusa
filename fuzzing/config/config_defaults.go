package config

import "github.com/trailofbits/medusa/compilation"

// GetDefaultProjectConfig obtains a default configuration for a project, given a platform.
func GetDefaultProjectConfig(platform string) (*ProjectConfig, error) {
	compilationConfig, err := compilation.NewCompilationConfig(platform)
	if err != nil {
		return nil, err
	}

	// Create a project configuration
	projectConfig := &ProjectConfig{
		Fuzzing: FuzzingConfig{
			Workers:                  10,
			WorkerDatabaseEntryLimit: 10000,
			Timeout:                  0,
			TestLimit:                0,
			MaxTxSequenceLength:      100,
			SenderAddresses: []string{
				"0x1111111111111111111111111111111111111111",
				"0x2222222222222222222222222222222222222222",
				"0x3333333333333333333333333333333333333333",
			},
			DeployerAddress: "0x1111111111111111111111111111111111111111",
			Testing: TestingConfig{
				StopOnFailedTest: true,
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
	}

	// Return the project configuration
	return projectConfig, nil
}
