package cmd

import (
	"github.com/trailofbits/medusa/compilation"
	"github.com/trailofbits/medusa/fuzzing/config"
)

// DefaultProjectConfigFilename describes the default config filename for a given project folder.
const DefaultProjectConfigFilename = "medusa.json"

// GetDefaultProjectConfig obtains a default configuration for a project, given a platform.
func GetDefaultProjectConfig(platform string) (*config.ProjectConfig, error) {
	compilationConfig, err := compilation.NewCompilationConfig(platform)
	if err != nil {
		return nil, err
	}

	// Create a project configuration
	projectConfig := &config.ProjectConfig{
		Fuzzing: config.FuzzingConfig{
			Workers:                  10,
			WorkerDatabaseEntryLimit: 10000,
			Timeout:                  0,
			TestLimit:                0,
			MaxTxSequenceLength:      100,
			PropertyTestPrefixes: []string{
				"fuzz_",
			},
			SenderAddresses: []string{
				"0x1111111111111111111111111111111111111111",
				"0x2222222222222222222222222222222222222222",
				"0x3333333333333333333333333333333333333333",
			},
			DeployerAddress: "0x1111111111111111111111111111111111111111",
		},
		Compilation: compilationConfig,
	}

	// Return the project configuration
	return projectConfig, nil
}
