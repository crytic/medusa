package cmd

import (
	"github.com/trailofbits/medusa/compilation"
	"github.com/trailofbits/medusa/configs"
)

// DefaultProjectConfigFilename describes the default config filename for a given project folder.
const DefaultProjectConfigFilename = "medusa.json"

// GetDefaultProjectConfig obtains a default configuration for a project, given a platform.
func GetDefaultProjectConfig(platform string) (*configs.ProjectConfig, error) {
	compilationConfig, err := compilation.GetDefaultCompilationConfig(platform)
	if err != nil {
		return nil, err
	}

	// Create a project configuration
	projectConfig := &configs.ProjectConfig{
		Accounts: configs.AccountConfig{
			Generate:   5,
			Predefined: make([]string, 0),
		},
		Fuzzing: configs.FuzzingConfig{
			Workers:                  10,
			WorkerDatabaseEntryLimit: 10000,
			Timeout:                  0,
			TestLimit:                0,
			MaxTxSequenceLength:      100,
			TestPrefixes: []string{
				"fuzz_",
			},
		},
		Compilation: *compilationConfig,
	}

	// Return the project configuration
	return projectConfig, nil
}
