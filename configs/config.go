package configs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type ProjectConfig struct {
	// Accounts offers configurable options for how many accounts to generate and allows user to provide existing keys.
	Accounts AccountConfig `json:"accounts"`

	// ThreadCount describes the amount of threads to use in fuzzing campaigns.
	ThreadCount int `json:"threads"`

	// CompilationSettings describes the configuration used to compile the underlying project.
	Compilation CompilationConfig `json:"compilation"`
}

type AccountConfig struct {
	// Generate describes how many accounts should be dynamically generated at runtime
	Generate int `json:"generate"`

	// Keys describe an existing set of keys that a user can provide to be used
	Keys []string `json:"keys,omitempty"`
}

type CompilationConfig struct {
	// Platform references an identifier indicating which compilation platform to use.
	// PlatformConfig is a structure dependent on the defined Platform.
	Platform string `json:"platform"`

	// PlatformConfig describes the Platform-specific configuration needed to compile.
	PlatformConfig *json.RawMessage `json:"platform_config"`
}

func ReadProjectConfigFromFile(path string) (*ProjectConfig, error) {
	// Read our project configuration file data
	fmt.Printf("Reading configuration file: %s\n", path)
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse the project configuration
	var projectConfig ProjectConfig
	err = json.Unmarshal(b, &projectConfig)
	if err != nil {
		return nil, err
	}
	return &projectConfig, nil
}

func (p *ProjectConfig) WriteToFile(path string) error {
	// Serialize the configuration
	b, err := json.MarshalIndent(p, "", "\t")
	if err != nil {
		return err
	}

	// Save it to the provided output path and return the result
	err = ioutil.WriteFile(path, b, 0644)
	if err != nil {
		return err
	}

	return nil
}