package config

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/trailofbits/medusa/chain/config"
	"github.com/trailofbits/medusa/compilation"
	"github.com/trailofbits/medusa/utils"
	"os"
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

	// WorkerResetLimit describes how many call sequences a worker should test before it is destroyed and recreated
	// so that memory from its underlying chain is freed.
	WorkerResetLimit int `json:"workerResetLimit"`

	// Timeout describes a time in seconds for which the fuzzing operation should run. Providing negative or zero value
	// will result in no timeout.
	Timeout int `json:"timeout"`

	// TestLimit describes a threshold for the number of transactions to test, after which it will exit. This number
	// must be non-negative. A zero value indicates the test limit should not be enforced.
	TestLimit uint64 `json:"testLimit"`

	// CallSequenceLength describes the maximum length a transaction sequence can be generated as.
	CallSequenceLength int `json:"callSequenceLength"`

	// CorpusDirectory describes the name for the folder that will hold the corpus and the coverage files. If empty,
	// the in-memory corpus will be used, but not flush to disk.
	CorpusDirectory string `json:"corpusDirectory"`

	// CoverageEnabled describes whether to use coverage-guided fuzzing
	CoverageEnabled bool `json:"coverageEnabled"`

	// DeploymentOrder determines the order in which the contracts should be deployed
	DeploymentOrder []string `json:"deploymentOrder"`

	// Constructor arguments for contracts deployment. It is available only in init mode
	ConstructorArgs map[string]map[string]any `json:"constructorArgs"`

	// DeployerAddress describe the account address to be used to deploy contracts.
	DeployerAddress string `json:"deployerAddress"`

	// SenderAddresses describe a set of account addresses to be used to send state-changing txs (calls) in fuzzing
	// campaigns.
	SenderAddresses []string `json:"senderAddresses"`

	// MaxBlockNumberDelay describes the maximum distance in block numbers the fuzzer will use when generating blocks
	// compared to the previous.
	MaxBlockNumberDelay uint64 `json:"blockNumberDelayMax"`

	// MaxBlockTimestampDelay describes the maximum distance in timestamps the fuzzer will use when generating blocks
	// compared to the previous.
	MaxBlockTimestampDelay uint64 `json:"blockTimestampDelayMax"`

	// BlockGasLimit describes the maximum amount of gas that can be used in a block by transactions. This defines
	// limits for how many transactions can be included per block.
	BlockGasLimit uint64 `json:"blockGasLimit"`

	// TransactionGasLimit describes the maximum amount of gas that will be used by the fuzzer generated transactions.
	TransactionGasLimit uint64 `json:"transactionGasLimit"`

	// Testing describes the configuration used for different testing strategies.
	Testing TestingConfig `json:"testingConfig"`

	// Logging describes the configuration used for logging
	Logging LoggingConfig `json:"loggingConfig"`

	// TestChain represents the chain.TestChain config to use when initializing a chain.
	TestChain config.TestChainConfig `json:"chainConfig"`
}

// TestingConfig describes the configuration options used for testing
type TestingConfig struct {
	// StopOnFailedTest describes whether the fuzzing.Fuzzer should stop after detecting the first failed test.
	StopOnFailedTest bool `json:"stopOnFailedTest"`

	// StopOnFailedContractMatching describes whether the fuzzing.Fuzzer should stop after failing to match bytecode
	// to determine which contract a deployed contract is.
	StopOnFailedContractMatching bool `json:"stopOnFailedContractMatching"`

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

// LoggingConfig describes the configuration options used for logging
type LoggingConfig struct {
	// Level describes whether logs of certain severity levels (eg info, warning, etc.) will be emitted or discarded.
	// Increasing level values represent more severe logs
	Level zerolog.Level `json:"level"`

	// EnableConsoleLogging describes whether console logging is enabled
	EnableConsoleLogging bool `json:"enableConsoleLogging"`

	// LogDirectory describes the directory where structured log _files_ will be outputted. If the string is empty, then
	// no log files are kept
	LogDirectory string `json:"logDirectory"`
}

// ReadProjectConfigFromFile reads a JSON-serialized ProjectConfig from a provided file path.
// Returns the ProjectConfig if it succeeds, or an error if one occurs.
func ReadProjectConfigFromFile(path string) (*ProjectConfig, error) {
	// Read our project configuration file data
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Parse the project configuration
	projectConfig, err := GetDefaultProjectConfig("")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, projectConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return projectConfig, nil
}

// WriteToFile writes the ProjectConfig to a provided file path in a JSON-serialized format.
// Returns an error if one occurs.
func (p *ProjectConfig) WriteToFile(path string) error {
	// Serialize the configuration
	b, err := json.MarshalIndent(p, "", "\t")
	if err != nil {
		return errors.WithStack(err)
	}

	// Save it to the provided output path and return the result
	err = os.WriteFile(path, b, 0644)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// Validate validates that the ProjectConfig meets certain requirements.
// Returns an error if one occurs.
func (p *ProjectConfig) Validate() error {
	// Verify the worker count is a positive number.
	if p.Fuzzing.Workers <= 0 {
		return errors.Errorf("fuzzer worker count must be positive number")
	}

	// Verify that the sequence length is a positive number
	if p.Fuzzing.CallSequenceLength <= 0 {
		return errors.Errorf("call sequence length must be a positive number")
	}

	// Verify the worker reset limit is a positive number
	if p.Fuzzing.WorkerResetLimit <= 0 {
		return errors.Errorf("worker reset limit must be a positive number")
	}

	// Verify gas limits are appropriate
	if p.Fuzzing.BlockGasLimit < p.Fuzzing.TransactionGasLimit {
		return errors.Errorf("block gas limit cannot be less than transaction gas limit")
	}
	if p.Fuzzing.BlockGasLimit == 0 || p.Fuzzing.TransactionGasLimit == 0 {
		return errors.Errorf("block and transaction gas limit cannot be zero")
	}

	// Verify that senders are well-formed addresses
	if _, err := utils.HexStringsToAddresses(p.Fuzzing.SenderAddresses); err != nil {
		return errors.Errorf("malformed sender address(es)")
	}

	// Verify that deployer is a well-formed address
	if _, err := utils.HexStringToAddress(p.Fuzzing.DeployerAddress); err != nil {
		return errors.Errorf("malformed deployer address")
	}

	// Verify property testing fields.
	if p.Fuzzing.Testing.PropertyTesting.Enabled {
		// Test prefixes must be supplied if property testing is enabled.
		if len(p.Fuzzing.Testing.PropertyTesting.TestPrefixes) == 0 {
			return errors.Errorf("must specify one or more test prefixes while in property mode")
		}
	}
	return nil
}
