package config

import "github.com/trailofbits/medusa/compilation"

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
			Workers:                  10,
			WorkerDatabaseEntryLimit: 10000,
			Timeout:                  0,
			TestLimit:                0,
			MaxTxSequenceLength:      100,
			DeploymentOrder:          []string{},
			CorpusDirectory:          "corpus",
			CoverageEnabled:          true,
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
