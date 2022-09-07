package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/trailofbits/medusa/compilation"
	"io/ioutil"
)

type ProjectConfig struct {
	// Fuzzing describes the configuration used in fuzzing campaigns.
	Fuzzing FuzzingConfig `json:"fuzzing"`

	// Compilation describes the configuration used to compile the underlying project.
	Compilation *compilation.CompilationConfig `json:"compilation"`
}

// FuzzingConfig describes the configuration options used by the fuzzing.Fuzzer.
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

	// DeployerAddress describe the account address to be used to deploy contracts.
	DeployerAddress string `json:"deployerAddress"`

	// SenderAddresses describe a set of account addresses to be used to send state-changing txs (calls) in fuzzing
	// campaigns.
	SenderAddresses []string `json:"senderAddresses"`

	// Testing describes the configuration used for different testing strategies.
	Testing TestingConfig `json:"testing"`
}

// TestingConfig describes the configuration options used for testing
type TestingConfig struct {
	// StopOnFailedTest describes whether the fuzzing.Fuzzer should stop after detecting the first failed test.
	StopOnFailedTest bool `json:"stopOnFailedTest"`

	// AssertionTesting describes the configuration used for assertion testing.
	AssertionTesting AssertionTestingConfig `json:"assertionTesting"`

	// PropertyTesting describes the configuration used for property testing.
	PropertyTesting PropertyTestConfig `json:"propertyTesting"`
}

// AssertionTestingConfig describes the configuration options used for assertion testing
type AssertionTestingConfig struct {
	// Enabled describes whether testing is enabled.
	Enabled bool `json:"enabled"`

	// TestViewMethods dictates whether constant/pure/view methods should be tested.
	TestViewMethods bool `json:"testViewMethods"`
}

// PropertyTestConfig describes the configuration options used for property testing
type PropertyTestConfig struct {
	// Enabled describes whether testing is enabled.
	Enabled bool `json:"enabled"`

	// TestPrefixes dictates what method name prefixes will determine if a contract method is a property test.
	TestPrefixes []string `json:"testPrefixes"`
}

// ReadProjectConfigFromFile reads a JSON-serialized ProjectConfig from a provided file path.
// Returns the ProjectConfig if it succeeds, or an error if one occurs.
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
	err = projectConfig.Validate()
	if err != nil {
		return nil, err
	}
	return &projectConfig, nil
}

// WriteToFile writes the ProjectConfig to a provided file path in a JSON-serialized format.
// Returns an error if one occurs.
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

// Validate validates that the ProjectConfig meets certain requirements.
// Returns an error if one occurs.
func (p *ProjectConfig) Validate() error {
	// Verify the worker count is a positive number.
	if p.Fuzzing.Workers <= 0 {
		return errors.New("project configuration must specify a positive number for the worker count")
	}

	// Verify property testing fields.
	if p.Fuzzing.Testing.PropertyTesting.Enabled {
		// Test prefixes must be supplied if property testing is enabled.
		if len(p.Fuzzing.Testing.PropertyTesting.TestPrefixes) == 0 {
			return errors.New("project configuration must specify test name prefixes if property testing is enabled")
		}
	}
	return nil
}
