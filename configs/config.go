package configs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
)

type ProjectConfig struct {
	// Accounts offers configurable options for how many accounts to generate and allows user to provide existing keys.
	Accounts AccountConfig `json:"accounts"`

	// Fuzzing describes the configuration used in fuzzing campaigns.
	Fuzzing FuzzingConfig `json:"fuzzing"`

	// Compilation describes the configuration used to compile the underlying project.
	Compilation CompilationConfig `json:"compilation"`
}

type AccountConfig struct {
	// Generate describes how many accounts should be dynamically generated at runtime
	Generate int `json:"generate"`

	// Predefined describe an existing set of accounts that a user can provide to be used
	Predefined []string `json:"predefined"`
}

type CompilationConfig struct {
	// Platform references an identifier indicating which compilation platform to use.
	// PlatformConfig is a structure dependent on the defined Platform.
	Platform string `json:"platform"`

	// PlatformConfig describes the Platform-specific configuration needed to compile.
	PlatformConfig *json.RawMessage `json:"platformConfig"`
}

type FuzzingConfig struct {
	// Workers describes the amount of threads to use in fuzzing campaigns.
	Workers int `json:"workers"`

	// WorkerDatabaseEntryLimit describes an upper bound on the amount of entries that can exist in a worker database
	// before it is pruned and recreated.
	WorkerDatabaseEntryLimit int `json:"workerDatabaseEntryLimit"`

	// Timeout describes a time in seconds for which the fuzzing operation should run. Providing negative or zero value
	// will result in no timeout.
	Timeout int `json:"timeout"`

	// TestLimit describes a threshold for the number of transactions to test, after which it will exit. This number
	// must be non-negative. A zero value indicates the test limit should not be enforced.
	TestLimit uint64 `json:"testLimit"`

	// MaxTxSequenceLength describes the maximum length a transaction sequence can be generated as.
	MaxTxSequenceLength int `json:"maxTxSequenceLength"`

	// TestPrefixes dictates what prefixes will determine that a fxn is a fuzz test
	// This can probably be moved to a different config struct once we isolate property testing from assertion testing
	TestPrefixes []string `json:"testPrefixes"`
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
	// Making validation a separate function in case we want to add other checks
	// The only check currently is to make sure at least one test prefix is provided
	err = projectConfig.ValidateConfig()
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

// ValidateConfig validate that the project config meets certain requirements
func (p *ProjectConfig) ValidateConfig() error {
	// Ensure at least one prefix is in config
	if len(p.Fuzzing.TestPrefixes) == 0 {
		return errors.New("at least one test prefix must be specified within your project configuration file")
	}
	return nil
}
