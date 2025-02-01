package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/crytic/medusa/compilation/types"
	"math/big"
	"os"

	"github.com/crytic/medusa/chain/config"
	"github.com/crytic/medusa/compilation"
	"github.com/crytic/medusa/logging"
	"github.com/crytic/medusa/utils"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog"
)

// The following directives will be picked up by the `go generate` command to generate JSON marshaling code from
// templates defined below. They should be preserved for re-use in case we change our structures.
//go:generate go get github.com/fjl/gencodec
//go:generate go run github.com/fjl/gencodec -type FuzzingConfig -field-override fuzzingConfigMarshaling -out gen_fuzzing_config.go

type ProjectConfig struct {
	// Fuzzing describes the configuration used in fuzzing campaigns.
	Fuzzing FuzzingConfig `json:"fuzzing"`

	// Compilation describes the configuration used to compile the underlying project.
	Compilation *compilation.CompilationConfig `json:"compilation"`

	// Slither describes the configuration for running slither
	Slither *types.SlitherConfig `json:"slither"`

	// Logging describes the configuration used for logging to file and console
	Logging LoggingConfig `json:"logging"`
}

// FuzzingConfig describes the configuration options used by the fuzzing.Fuzzer.
type FuzzingConfig struct {
	// Workers describes the amount of threads to use in fuzzing campaigns.
	Workers int `json:"workers"`

	// WorkerResetLimit describes how many call sequences a worker should test before it is destroyed and recreated
	// so that memory from its underlying chain is freed.
	WorkerResetLimit int `json:"workerResetLimit"`

	// Timeout describes a time threshold in seconds for which the fuzzing operation should run. Providing negative or
	// zero value will result in no timeout.
	Timeout int `json:"timeout"`

	// TestLimit describes a threshold for the number of transactions to test, after which it will exit. This number
	// must be non-negative. A zero value indicates the test limit should not be enforced.
	TestLimit uint64 `json:"testLimit"`

	// ShrinkLimit describes a threshold for the iterations (call sequence tests) which shrinking should perform.
	ShrinkLimit uint64 `json:"shrinkLimit"`

	// CallSequenceLength describes the maximum length a transaction sequence can be generated as.
	CallSequenceLength int `json:"callSequenceLength"`

	// CorpusDirectory describes the name for the folder that will hold the corpus and the coverage files. If empty,
	// the in-memory corpus will be used, but not flush to disk.
	CorpusDirectory string `json:"corpusDirectory"`

	// CoverageEnabled describes whether to use coverage-guided fuzzing
	CoverageEnabled bool `json:"coverageEnabled"`

	// CoverageFormats indicate which reports to generate: "lcov" and "html" are supported.
	CoverageFormats []string `json:"coverageFormats"`

	// TargetContracts are the target contracts for fuzz testing
	TargetContracts []string `json:"targetContracts"`

	// PredeployedContracts are contracts that can be deterministically deployed at a specific address. It maps the
	// contract name to the deployment address
	PredeployedContracts map[string]string `json:"predeployedContracts"`

	// TargetContractsBalances holds the amount of wei that should be sent during deployment for one or more contracts in
	// TargetContracts
	TargetContractsBalances []*big.Int `json:"targetContractsBalances"`

	// ConstructorArgs holds the constructor arguments for TargetContracts deployments. It is available via the project
	// configuration
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

	// ReversionReporterEnabled determines whether reversion metrics should be collected and reported.
	ReversionReporterEnabled bool `json:"reversionReporterEnabled"`

	// Testing describes the configuration used for different testing strategies.
	Testing TestingConfig `json:"testing"`

	// TestChainConfig represents the chain.TestChain config to use when initializing a chain.
	TestChainConfig config.TestChainConfig `json:"chainConfig"`
}

// fuzzingConfigMarshaling is a structure that overrides field types during JSON marshaling. It allows FuzzingConfig to
// have its custom marshaling methods auto-generated and will handle type conversions for serialization purposes.
// For example, this enables serialization of big.Int but specifying a different field type to control serialization.
type fuzzingConfigMarshaling struct {
	TargetContractsBalances []*hexutil.Big
}

// TestingConfig describes the configuration options used for testing
type TestingConfig struct {
	// StopOnFailedTest describes whether the fuzzing.Fuzzer should stop after detecting the first failed test.
	StopOnFailedTest bool `json:"stopOnFailedTest"`

	// StopOnFailedContractMatching describes whether the fuzzing.Fuzzer should stop after failing to match bytecode
	// to determine which contract a deployed contract is.
	StopOnFailedContractMatching bool `json:"stopOnFailedContractMatching"`

	// StopOnNoTests describes whether the fuzzing.Fuzzer should stop the fuzzer from starting if no tests (property,
	// assertion, optimization, custom) are found.
	StopOnNoTests bool `json:"stopOnNoTests"`

	// TestAllContracts indicates whether all contracts should be tested (including dynamically deployed ones), rather
	// than just the contracts specified in the project configuration's deployment order.
	TestAllContracts bool `json:"testAllContracts"`

	// TraceAll describes whether a trace should be attached to each element of a finalized shrunken call sequence,
	// e.g. when a call sequence triggers a test failure. Test providers may attach execution traces by default,
	// even if this option is not enabled.
	TraceAll bool `json:"traceAll"`

	// AssertionTesting describes the configuration used for assertion testing.
	AssertionTesting AssertionTestingConfig `json:"assertionTesting"`

	// PropertyTesting describes the configuration used for property testing.
	PropertyTesting PropertyTestingConfig `json:"propertyTesting"`

	// OptimizationTesting describes the configuration used for optimization testing.
	OptimizationTesting OptimizationTestingConfig `json:"optimizationTesting"`

	// TargetFunctionSignatures is a list function signatures call the fuzzer should exclusively target by omitting calls to other signatures.
	// The signatures should specify the contract name and signature in the ABI format like `Contract.func(uint256,bytes32)`.
	TargetFunctionSignatures []string `json:"targetFunctionSignatures"`

	// ExcludeFunctionSignatures is a list of function signatures that will be excluded from call sequences.
	// The signatures should specify the contract name and signature in the ABI format like `Contract.func(uint256,bytes32)`.
	ExcludeFunctionSignatures []string `json:"excludeFunctionSignatures"`
}

// Validate validates that the TestingConfig meets certain requirements.
func (testCfg *TestingConfig) Validate() error {
	// Verify that target and exclude function signatures are used mutually exclusive.
	if (len(testCfg.TargetFunctionSignatures) != 0) && (len(testCfg.ExcludeFunctionSignatures) != 0) {
		return errors.New("project configuration must specify only one of blacklist or whitelist at a time")
	}

	// Verify property testing fields.
	if testCfg.PropertyTesting.Enabled {
		// Test prefixes must be supplied if property testing is enabled.
		if len(testCfg.PropertyTesting.TestPrefixes) == 0 {
			return errors.New("project configuration must specify test name prefixes if property testing is enabled")
		}
	}

	if testCfg.OptimizationTesting.Enabled {
		// Test prefixes must be supplied if optimization testing is enabled.
		if len(testCfg.OptimizationTesting.TestPrefixes) == 0 {
			return errors.New("project configuration must specify test name prefixes if optimization testing is enabled")
		}
	}

	// Validate that prefixes do not overlap
	for _, prefix := range testCfg.PropertyTesting.TestPrefixes {
		for _, prefix2 := range testCfg.OptimizationTesting.TestPrefixes {
			if prefix == prefix2 {
				return errors.New("project configuration must specify unique test name prefixes for property and optimization testing")
			}
		}
	}

	return nil
}

// AssertionTestingConfig describes the configuration options used for assertion testing
type AssertionTestingConfig struct {
	// Enabled describes whether testing is enabled.
	Enabled bool `json:"enabled"`

	// TestViewMethods dictates whether constant/pure/view methods should be tested.
	TestViewMethods bool `json:"testViewMethods"`

	// PanicCodeConfig describes the various panic codes that can be enabled and be treated as a "failing case"
	PanicCodeConfig PanicCodeConfig `json:"panicCodeConfig"`
}

// PanicCodeConfig describes the various panic codes that can be enabled and be treated as a failing assertion test
type PanicCodeConfig struct {
	// FailOnCompilerInsertedPanic describes whether a generic compiler inserted panic should be treated as a failing case
	FailOnCompilerInsertedPanic bool `json:"failOnCompilerInsertedPanic"`

	// FailOnAssertion describes whether an assertion failure should be treated as a failing case
	FailOnAssertion bool `json:"failOnAssertion"`

	// FailOnArithmeticUnderflow describes whether an arithmetic underflow should be treated as a failing case
	FailOnArithmeticUnderflow bool `json:"failOnArithmeticUnderflow"`

	// FailOnDivideByZero describes whether division by zero should be treated as a failing case
	FailOnDivideByZero bool `json:"failOnDivideByZero"`

	// FailOnEnumTypeConversionOutOfBounds describes whether an out-of-bounds enum access should be treated as a failing case
	FailOnEnumTypeConversionOutOfBounds bool `json:"failOnEnumTypeConversionOutOfBounds"`

	// FailOnIncorrectStorageAccess describes whether an out-of-bounds storage access should be treated as a failing case
	FailOnIncorrectStorageAccess bool `json:"failOnIncorrectStorageAccess"`

	// FailOnPopEmptyArray describes whether a pop operation on an empty array should be treated as a failing case
	FailOnPopEmptyArray bool `json:"failOnPopEmptyArray"`

	// FailOnOutOfBoundsArrayAccess describes whether an out-of-bounds array access should be treated as a failing case
	FailOnOutOfBoundsArrayAccess bool `json:"failOnOutOfBoundsArrayAccess"`

	// FailOnAllocateTooMuchMemory describes whether excessive memory usage should be treated as a failing case
	FailOnAllocateTooMuchMemory bool `json:"failOnAllocateTooMuchMemory"`

	// FailOnCallUninitializedVariable describes whether calling an un-initialized variable should be treated as a failing case
	FailOnCallUninitializedVariable bool `json:"failOnCallUninitializedVariable"`
}

// PropertyTestingConfig describes the configuration options used for property testing
type PropertyTestingConfig struct {
	// Enabled describes whether testing is enabled.
	Enabled bool `json:"enabled"`

	// TestPrefixes dictates what method name prefixes will determine if a contract method is a property test.
	TestPrefixes []string `json:"testPrefixes"`
}

// OptimizationTestingConfig describes the configuration options used for optimization testing
type OptimizationTestingConfig struct {
	// Enabled describes whether testing is enabled.
	Enabled bool `json:"enabled"`

	// TestPrefixes dictates what method name prefixes will determine if a contract method is an optimization test.
	TestPrefixes []string `json:"testPrefixes"`
}

// LoggingConfig describes the configuration options for logging to console and file
type LoggingConfig struct {
	// Level describes whether logs of certain severity levels (eg info, warning, etc.) will be emitted or discarded.
	// Increasing level values represent more severe logs
	Level zerolog.Level `json:"level"`

	// LogDirectory describes what directory log files should be outputted in/ LogDirectory being a non-empty string is
	// equivalent to enabling file logging.
	LogDirectory string `json:"logDirectory"`

	// NoColor indicates whether log messages should be displayed with colored formatting.
	NoColor bool `json:"noColor"`
}

// ConsoleLoggingConfig describes the configuration options for logging to console. Note that this not being used right now
// but will be added to LoggingConfig down the line
// TODO: Update when implementing a structured logging solution
type ConsoleLoggingConfig struct {
	// Enabled describes whether console logging is enabled.
	Enabled bool `json:"enabled"`
}

// FileLoggingConfig describes the configuration options for logging to file. Note that this not being used right now
// but will be added to LoggingConfig down the line
// TODO: Update when implementing a structured logging solution
type FileLoggingConfig struct {
	// LogDirectory describes what directory log files should be outputted in. LogDirectory being a non-empty string
	// is equivalent to enabling file logging.
	LogDirectory bool `json:"logDirectory"`
}

// ReadProjectConfigFromFile reads a JSON-serialized ProjectConfig from a provided file path.
// Returns the ProjectConfig if it succeeds, or an error if one occurs.
func ReadProjectConfigFromFile(path string, platform string) (*ProjectConfig, error) {
	// Read our project configuration file data
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse the project configuration
	projectConfig, err := GetDefaultProjectConfig(platform)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, projectConfig)
	if err != nil {
		return nil, err
	}

	return projectConfig, nil
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
	err = os.WriteFile(path, b, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Validate validates that the ProjectConfig meets certain requirements.
// Returns an error if one occurs.
func (p *ProjectConfig) Validate() error {
	// Create logger instance if global logger is available
	logger := logging.NewLogger(zerolog.Disabled)
	if logging.GlobalLogger != nil {
		logger = logging.GlobalLogger.NewSubLogger("module", "fuzzer config")
	}

	// Validate testing config
	if err := p.Fuzzing.Testing.Validate(); err != nil {
		return err
	}

	// Verify the worker count is a positive number.
	if p.Fuzzing.Workers <= 0 {
		return errors.New("project configuration must specify a positive number for the worker count")
	}

	// Verify that the sequence length is a positive number
	if p.Fuzzing.CallSequenceLength <= 0 {
		return errors.New("project configuration must specify a positive number for the transaction sequence length")
	}

	// Verify the worker reset limit is a positive number
	if p.Fuzzing.WorkerResetLimit <= 0 {
		return errors.New("project configuration must specify a positive number for the worker reset limit")
	}

	// Verify timeout
	if p.Fuzzing.Timeout < 0 {
		return errors.New("project configuration must specify a positive number for the timeout")
	}

	// Verify gas limits are appropriate
	if p.Fuzzing.BlockGasLimit < p.Fuzzing.TransactionGasLimit {
		return errors.New("project configuration must specify a block gas limit which is not less than the transaction gas limit")
	}
	if p.Fuzzing.BlockGasLimit == 0 || p.Fuzzing.TransactionGasLimit == 0 {
		return errors.New("project configuration must specify a block and transaction gas limit which are non-zero")
	}

	// Log warning if max block delay is zero
	if p.Fuzzing.MaxBlockNumberDelay == 0 {
		logger.Warn("The maximum block number delay is set to zero. Please be aware that transactions will " +
			"always be fit in the same block until the block gas limit is reached and that the block number will always " +
			"increment by one.")
	}

	// Log warning if max timestamp delay is zero
	if p.Fuzzing.MaxBlockTimestampDelay == 0 {
		logger.Warn("The maximum timestamp delay is set to zero. Please be aware that block time jumps will " +
			"always be exactly one.")
	}

	// Verify that senders are well-formed addresses
	if _, err := utils.HexStringsToAddresses(p.Fuzzing.SenderAddresses); err != nil {
		return errors.New("project configuration must specify only well-formed sender address(es)")
	}

	// Verify that deployer is a well-formed address
	if _, err := utils.HexStringToAddress(p.Fuzzing.DeployerAddress); err != nil {
		return errors.New("project configuration must specify only a well-formed deployer address")
	}

	// Verify that addresses of predeployed contracts are well-formed
	for _, addr := range p.Fuzzing.PredeployedContracts {
		if _, err := utils.HexStringToAddress(addr); err != nil {
			return errors.New("project configuration must specify only well-formed predeployed contract address(es)")
		}
	}

	// The coverage report format must be either "lcov" or "html"
	if p.Fuzzing.CoverageFormats != nil {
		for _, report := range p.Fuzzing.CoverageFormats {
			if report != "lcov" && report != "html" {
				return fmt.Errorf("project configuration must specify only valid coverage reports (lcov, html): %s", report)
			}
		}
	}

	// Ensure that the log level is a valid one
	level, err := zerolog.ParseLevel(p.Logging.Level.String())
	if err != nil || level == zerolog.FatalLevel {
		return errors.New("project config must specify a valid log level (trace, debug, info, warn, error, or panic)")
	}

	return nil
}
