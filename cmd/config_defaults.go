package cmd

import (
	"medusa/compilation"
	"medusa/configs"
)

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
			Generate: 5,
		},
		Fuzzing: configs.FuzzingConfig{
			Workers: 10,
			MaxTxSequenceLength: 10,
		},
		Compilation: *compilationConfig,
	}

	// Return the project configuration
	return projectConfig, nil
}
