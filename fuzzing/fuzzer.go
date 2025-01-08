package fuzzing

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/crytic/medusa/fuzzing/executiontracer"

	"github.com/crytic/medusa/fuzzing/coverage"
	"github.com/crytic/medusa/logging"
	"github.com/crytic/medusa/logging/colors"
	"github.com/rs/zerolog"

	"github.com/crytic/medusa/fuzzing/calls"
	"github.com/crytic/medusa/utils/randomutils"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/crytic/medusa/chain"
	compilationTypes "github.com/crytic/medusa/compilation/types"
	"github.com/crytic/medusa/fuzzing/config"
	fuzzerTypes "github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/fuzzing/corpus"
	fuzzingutils "github.com/crytic/medusa/fuzzing/utils"
	"github.com/crytic/medusa/fuzzing/valuegeneration"
	"github.com/crytic/medusa/utils"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/exp/slices"
)

// Fuzzer represents an Ethereum smart contract fuzzing provider.
type Fuzzer struct {
	// ctx describes the context for the fuzzing run, used to cancel running operations.
	ctx context.Context
	// ctxCancelFunc describes a function which can be used to cancel the fuzzing operations ctx tracks.
	ctxCancelFunc context.CancelFunc

	// config describes the project configuration which the fuzzing is targeting.
	config config.ProjectConfig
	// senders describes a set of account addresses used to send state changing calls in fuzzing campaigns.
	senders []common.Address
	// deployer describes an account address used to deploy contracts in fuzzing campaigns.
	deployer common.Address

	// compilations describes all compilations added as targets.
	compilations []compilationTypes.Compilation
	// contractDefinitions defines targets to be fuzzed once their deployment is detected. They are derived from
	// compilations.
	contractDefinitions fuzzerTypes.Contracts
	// slitherResults holds the results obtained from slither. At the moment we do not have use for storing this in the
	// Fuzzer but down the line we can use slither for other capabilities that may require storage of the results.
	slitherResults *compilationTypes.SlitherResults

	// baseValueSet represents a valuegeneration.ValueSet containing input values for our fuzz tests.
	baseValueSet *valuegeneration.ValueSet

	// workers represents the work threads created by this Fuzzer when Start invokes a fuzz operation.
	workers []*FuzzerWorker
	// metrics represents the metrics for the fuzzing campaign.
	metrics *FuzzerMetrics
	// corpus stores a list of transaction sequences that can be used for coverage-guided fuzzing
	corpus *corpus.Corpus

	// randomProvider describes the provider used to generate random values in the Fuzzer. All other random providers
	// used by the Fuzzer's subcomponents are derived from this one.
	randomProvider *rand.Rand

	// testCases contains every TestCase registered with the Fuzzer.
	testCases []TestCase
	// testCasesLock provides thread-synchronization to avoid race conditions when accessing or updating test cases.
	testCasesLock sync.Mutex
	// testCasesFinished describes test cases already reported as having been finalized.
	testCasesFinished map[string]TestCase

	// Events describes the event system for the Fuzzer.
	Events FuzzerEvents

	// Hooks describes the replaceable functions used by the Fuzzer.
	Hooks FuzzerHooks

	// logger describes the Fuzzer's log object that can be used to log important events
	logger *logging.Logger
}

// NewFuzzer returns an instance of a new Fuzzer provided a project configuration, or an error if one is encountered
// while initializing the code.
func NewFuzzer(config config.ProjectConfig) (*Fuzzer, error) {
	// Disable colors if requested
	if config.Logging.NoColor {
		colors.DisableColor()
	}

	// Create the global logger and add stdout as an unstructured output stream
	// Note that we are not using the project config's log level because we have not validated it yet
	logging.GlobalLogger = logging.NewLogger(config.Logging.Level)
	logging.GlobalLogger.AddWriter(os.Stdout, logging.UNSTRUCTURED, !config.Logging.NoColor)

	// If the log directory is a non-empty string, create a file for unstructured, un-colorized file logging
	if config.Logging.LogDirectory != "" {
		// Filename will be the "log-current_unix_timestamp.log"
		filename := "log-" + strconv.FormatInt(time.Now().Unix(), 10) + ".log"
		// Create the file
		file, err := utils.CreateFile(config.Logging.LogDirectory, filename)
		if err != nil {
			logging.GlobalLogger.Error("Failed to create log file", err)
			return nil, err
		}
		logging.GlobalLogger.AddWriter(file, logging.UNSTRUCTURED, false)
	}

	// Validate our provided config
	err := config.Validate()
	if err != nil {
		logging.GlobalLogger.Error("Invalid configuration", err)
		return nil, err
	}

	// Update the log level of the global logger now
	logging.GlobalLogger.SetLevel(config.Logging.Level)

	// Get the fuzzer's custom sub-logger
	logger := logging.GlobalLogger.NewSubLogger("module", "fuzzer")

	// Parse the senders addresses from our account config.
	senders, err := utils.HexStringsToAddresses(config.Fuzzing.SenderAddresses)
	if err != nil {
		logger.Error("Invalid sender address(es)", err)
		return nil, err
	}

	// Parse the deployer address from our account config
	deployer, err := utils.HexStringToAddress(config.Fuzzing.DeployerAddress)
	if err != nil {
		logger.Error("Invalid deployer address", err)
		return nil, err
	}

	// Create and return our fuzzing instance.
	fuzzer := &Fuzzer{
		config:              config,
		senders:             senders,
		deployer:            deployer,
		baseValueSet:        valuegeneration.NewValueSet(),
		contractDefinitions: make(fuzzerTypes.Contracts, 0),
		testCases:           make([]TestCase, 0),
		testCasesFinished:   make(map[string]TestCase),
		Hooks: FuzzerHooks{
			NewCallSequenceGeneratorConfigFunc: defaultCallSequenceGeneratorConfigFunc,
			NewShrinkingValueMutatorFunc:       defaultShrinkingValueMutatorFunc,
			ChainSetupFunc:                     chainSetupFromCompilations,
			CallSequenceTestFuncs:              make([]CallSequenceTestFunc, 0),
		},
		logger: logger,
	}

	// Add our sender and deployer addresses to the base value set for the value generator, so they will be used as
	// address arguments in fuzzing campaigns.
	fuzzer.baseValueSet.AddAddress(fuzzer.deployer)
	for _, sender := range fuzzer.senders {
		fuzzer.baseValueSet.AddAddress(sender)
	}

	// If we have a compilation config
	if fuzzer.config.Compilation != nil {
		// Compile the targets specified in the compilation config
		fuzzer.logger.Info("Compiling targets with ", colors.Bold, fuzzer.config.Compilation.Platform, colors.Reset)
		start := time.Now()
		compilations, _, err := (*fuzzer.config.Compilation).Compile()
		if err != nil {
			fuzzer.logger.Error("Failed to compile target", err)
			return nil, err
		}
		fuzzer.logger.Info("Finished compiling targets in ", time.Since(start).Round(time.Second))

		// Add our compilation targets
		fuzzer.AddCompilationTargets(compilations)
	}

	// Register any default providers if specified.
	if fuzzer.config.Fuzzing.Testing.PropertyTesting.Enabled {
		attachPropertyTestCaseProvider(fuzzer)
	}
	if fuzzer.config.Fuzzing.Testing.AssertionTesting.Enabled {
		attachAssertionTestCaseProvider(fuzzer)
	}
	if fuzzer.config.Fuzzing.Testing.OptimizationTesting.Enabled {
		attachOptimizationTestCaseProvider(fuzzer)
	}
	return fuzzer, nil
}

// ContractDefinitions exposes the contract definitions registered with the Fuzzer.
func (f *Fuzzer) ContractDefinitions() fuzzerTypes.Contracts {
	return slices.Clone(f.contractDefinitions)
}

// Config exposes the underlying project configuration provided to the Fuzzer.
func (f *Fuzzer) Config() config.ProjectConfig {
	return f.config
}

// BaseValueSet exposes the underlying value set provided to the Fuzzer value generators to aid in generation
// (e.g. for use in mutation operations).
func (f *Fuzzer) BaseValueSet() *valuegeneration.ValueSet {
	return f.baseValueSet
}

// SenderAddresses exposes the account addresses from which state changing fuzzed transactions will be sent by a
// FuzzerWorker.
func (f *Fuzzer) SenderAddresses() []common.Address {
	return f.senders
}

// DeployerAddress exposes the account address from which contracts will be deployed by a FuzzerWorker.
func (f *Fuzzer) DeployerAddress() common.Address {
	return f.deployer
}

// TestCases exposes the underlying tests run during the fuzzing campaign.
func (f *Fuzzer) TestCases() []TestCase {
	return f.testCases
}

// TestCasesWithStatus exposes the underlying tests with the provided status.
func (f *Fuzzer) TestCasesWithStatus(status TestCaseStatus) []TestCase {
	// Acquire a thread lock to avoid race conditions
	f.testCasesLock.Lock()
	defer f.testCasesLock.Unlock()

	// Collect all test cases with matching statuses.
	return utils.SliceWhere(f.testCases, func(t TestCase) bool {
		return t.Status() == status
	})
}

// RegisterTestCase registers a new TestCase with the Fuzzer.
func (f *Fuzzer) RegisterTestCase(testCase TestCase) {
	// Acquire a thread lock to avoid race conditions
	f.testCasesLock.Lock()
	defer f.testCasesLock.Unlock()

	// Display what is being tested
	f.logger.Info(testCase.LogMessage().Elements()...)

	// Append our test case to our list
	f.testCases = append(f.testCases, testCase)
}

// ReportTestCaseFinished is used to report a TestCase status as finalized to the Fuzzer.
func (f *Fuzzer) ReportTestCaseFinished(testCase TestCase) {
	// Acquire a thread lock to avoid race conditions
	f.testCasesLock.Lock()
	defer f.testCasesLock.Unlock()

	// If we already reported this test case as finished, stop
	if _, alreadyExists := f.testCasesFinished[testCase.ID()]; alreadyExists {
		return
	}

	// Otherwise now mark the test case as finished.
	f.testCasesFinished[testCase.ID()] = testCase

	// We only log here if we're not configured to stop on the first test failure. This is because the fuzzer prints
	// results on exit, so we avoid duplicate messages.
	if !f.config.Fuzzing.Testing.StopOnFailedTest {
		f.logger.Info(testCase.LogMessage().Elements()...)
	}

	// If the config specifies, we stop after the first failed test reported.
	if testCase.Status() == TestCaseStatusFailed && f.config.Fuzzing.Testing.StopOnFailedTest {
		f.Stop()
	}
}

// AddCompilationTargets takes a compilation and updates the Fuzzer state with additional Fuzzer.ContractDefinitions
// definitions and Fuzzer.BaseValueSet values.
func (f *Fuzzer) AddCompilationTargets(compilations []compilationTypes.Compilation) {
	var seedFromAST bool

	// No need to handle the error here since having compilation artifacts implies that we used a supported
	// platform configuration
	platformConfig, _ := f.config.Compilation.GetPlatformConfig()

	// Retrieve the compilation target for slither
	target := platformConfig.GetTarget()

	// Run slither and handle errors
	slitherResults, err := f.config.Slither.RunSlither(target)
	if err != nil || slitherResults == nil {
		if err != nil {
			f.logger.Warn("Failed to run slither", err)
		}
		seedFromAST = true
	}

	// If we have results and there were no errors, we will seed the value set using the slither results
	if !seedFromAST {
		f.slitherResults = slitherResults
		// Seed our base value set with the constants extracted by Slither
		f.baseValueSet.SeedFromSlither(slitherResults)
	}

	// Capture all the contract definitions, functions, and cache the source code
	for i := 0; i < len(compilations); i++ {
		// Add our compilation to the list and get a reference to it.
		f.compilations = append(f.compilations, compilations[i])
		compilation := &f.compilations[len(f.compilations)-1]

		// Loop for each source
		for sourcePath, source := range compilation.SourcePathToArtifact {
			// Seed from the contract's AST if we did not use slither or failed to do so
			if seedFromAST {
				// Seed our base value set from every source's AST
				f.baseValueSet.SeedFromAst(source.Ast)
			}

			// Loop for every contract and register it in our contract definitions
			for contractName := range source.Contracts {
				contract := source.Contracts[contractName]

				// Skip interfaces.
				if contract.Kind == compilationTypes.ContractKindInterface {
					continue
				}

				contractDefinition := fuzzerTypes.NewContract(contractName, sourcePath, &contract, compilation)

				// Sort available methods by type
				assertionTestMethods, propertyTestMethods, optimizationTestMethods := fuzzingutils.BinTestByType(&contract,
					f.config.Fuzzing.Testing.PropertyTesting.TestPrefixes,
					f.config.Fuzzing.Testing.OptimizationTesting.TestPrefixes,
					f.config.Fuzzing.Testing.AssertionTesting.TestViewMethods)
				contractDefinition.AssertionTestMethods = assertionTestMethods
				contractDefinition.PropertyTestMethods = propertyTestMethods
				contractDefinition.OptimizationTestMethods = optimizationTestMethods

				// Filter and record methods available for assertion testing. Property and optimization tests are always run.
				if len(f.config.Fuzzing.Testing.TargetFunctionSignatures) > 0 {
					// Only consider methods that are in the target methods list
					contractDefinition = contractDefinition.WithTargetedAssertionMethods(f.config.Fuzzing.Testing.TargetFunctionSignatures)
				}
				if len(f.config.Fuzzing.Testing.ExcludeFunctionSignatures) > 0 {
					// Consider all methods except those in the exclude methods list
					contractDefinition = contractDefinition.WithExcludedAssertionMethods(f.config.Fuzzing.Testing.ExcludeFunctionSignatures)
				}

				f.contractDefinitions = append(f.contractDefinitions, contractDefinition)
			}
		}

		// Cache all of our source code if it hasn't been already.
		err := compilation.CacheSourceCode()
		if err != nil {
			f.logger.Warn("Failed to cache compilation source file data", err)
		}
	}
}

// createTestChain creates a test chain with the account balance allocations specified by the config.
func (f *Fuzzer) createTestChain() (*chain.TestChain, error) {
	// Create our genesis allocations.
	// NOTE: Sharing GenesisAlloc between chains will result in some accounts not being funded for some reason.
	genesisAlloc := make(types.GenesisAlloc)

	// Fund all of our sender addresses in the genesis block
	initBalance := new(big.Int).Div(abi.MaxInt256, big.NewInt(2)) // TODO: make this configurable
	for _, sender := range f.senders {
		genesisAlloc[sender] = types.Account{
			Balance: initBalance,
		}
	}

	// Fund our deployer address in the genesis block
	genesisAlloc[f.deployer] = types.Account{
		Balance: initBalance,
	}

	// Identify which contracts need to be predeployed to a deterministic address by iterating across the mapping
	contractAddressOverrides := make(map[common.Hash]common.Address, len(f.config.Fuzzing.PredeployedContracts))
	for contractName, addrStr := range f.config.Fuzzing.PredeployedContracts {
		found := false
		// Try to find the associated compilation artifact
		for _, contract := range f.contractDefinitions {
			if contract.Name() == contractName {
				// Hash the init bytecode (so that it can be easily identified in the EVM) and map it to the
				// requested address
				initBytecodeHash := crypto.Keccak256Hash(contract.CompiledContract().InitBytecode)
				contractAddr, err := utils.HexStringToAddress(addrStr)
				if err != nil {
					return nil, fmt.Errorf("invalid address provided for a predeployed contract: %v", contract.Name())
				}
				contractAddressOverrides[initBytecodeHash] = contractAddr
				found = true
				break
			}
		}

		// Throw an error if the contract specified in the config is not found
		if !found {
			return nil, fmt.Errorf("%v was specified in the predeployed contracts but was not found in the compilation artifacts", contractName)
		}
	}

	// Update the test chain config with the contract address overrides
	f.config.Fuzzing.TestChainConfig.ContractAddressOverrides = contractAddressOverrides

	// Create our test chain with our basic allocations and passed medusa's chain configuration
	testChain, err := chain.NewTestChain(genesisAlloc, &f.config.Fuzzing.TestChainConfig)

	// Set our block gas limit
	testChain.BlockGasLimit = f.config.Fuzzing.BlockGasLimit
	return testChain, err
}

// chainSetupFromCompilations is a TestChainSetupFunc which sets up the base test chain state by deploying
// all compiled contract definitions. This includes any successful compilations as a result of the Fuzzer.config
// definitions, as well as those added by Fuzzer.AddCompilationTargets. The contract deployment order is defined by
// the Fuzzer.config.
func chainSetupFromCompilations(fuzzer *Fuzzer, testChain *chain.TestChain) (*executiontracer.ExecutionTrace, error) {
	// Verify that target contracts is not empty. If it's empty, but we only have one contract definition,
	// we can infer the target contracts. Otherwise, we report an error.
	if len(fuzzer.config.Fuzzing.TargetContracts) == 0 {
		var found bool
		for _, contract := range fuzzer.contractDefinitions {
			// If only one contract is defined, we can infer the target contract by filtering interfaces/libraries.
			if contract.CompiledContract().Kind == compilationTypes.ContractKindContract {
				if !found {
					fuzzer.config.Fuzzing.TargetContracts = []string{contract.Name()}
					found = true
				} else {
					// TODO list options for the user to choose from
					return nil, fmt.Errorf("specify target contract(s)")
				}
			}
		}
	}

	// Concatenate the predeployed contracts and target contracts
	// Ordering is important here (predeploys _then_ targets) so that you can have the same contract in both lists
	// while still being able to use the contract address overrides
	contractsToDeploy := make([]string, 0)
	balances := make([]*big.Int, 0)
	for contractName := range fuzzer.config.Fuzzing.PredeployedContracts {
		contractsToDeploy = append(contractsToDeploy, contractName)
		// Preserve index of target contract balances
		balances = append(balances, big.NewInt(0))
	}
	contractsToDeploy = append(contractsToDeploy, fuzzer.config.Fuzzing.TargetContracts...)
	balances = append(balances, fuzzer.config.Fuzzing.TargetContractsBalances...)

	deployedContractAddr := make(map[string]common.Address)
	// Loop for all contracts to deploy
	for i, contractName := range contractsToDeploy {
		// Look for a contract in our compiled contract definitions that matches this one
		found := false
		for _, contract := range fuzzer.contractDefinitions {
			// If we found a contract definition that matches this definition by name, try to deploy it
			if contract.Name() == contractName {
				// Concatenate constructor arguments, if necessary
				args := make([]any, 0)
				if len(contract.CompiledContract().Abi.Constructor.Inputs) > 0 {
					// If the contract is a predeployed contract, throw an error because they do not accept constructor
					// args.
					if _, ok := fuzzer.config.Fuzzing.PredeployedContracts[contractName]; ok {
						return nil, fmt.Errorf("predeployed contracts cannot accept constructor arguments")
					}
					jsonArgs, ok := fuzzer.config.Fuzzing.ConstructorArgs[contractName]
					if !ok {
						return nil, fmt.Errorf("constructor arguments for contract %s not provided", contractName)
					}
					decoded, err := valuegeneration.DecodeJSONArgumentsFromMap(contract.CompiledContract().Abi.Constructor.Inputs,
						jsonArgs, deployedContractAddr)
					if err != nil {
						return nil, err
					}
					args = decoded
				}

				// Construct our deployment message/tx data field
				msgData, err := contract.CompiledContract().GetDeploymentMessageData(args)
				if err != nil {
					return nil, fmt.Errorf("initial contract deployment failed for contract \"%v\", error: %v", contractName, err)
				}

				// If our project config has a non-zero balance for this target contract, retrieve it
				contractBalance := big.NewInt(0)
				if len(balances) > i {
					contractBalance = new(big.Int).Set(balances[i])
				}

				// Create a message to represent our contract deployment (we let deployments consume the whole block
				// gas limit rather than use tx gas limit)
				msg := calls.NewCallMessage(fuzzer.deployer, nil, 0, contractBalance, fuzzer.config.Fuzzing.BlockGasLimit, nil, nil, nil, msgData)
				msg.FillFromTestChainProperties(testChain)

				// Create a new pending block we'll commit to chain
				block, err := testChain.PendingBlockCreate()
				if err != nil {
					return nil, err
				}

				// Add our transaction to the block
				err = testChain.PendingBlockAddTx(msg.ToCoreMessage())
				if err != nil {
					return nil, err
				}

				// Commit the pending block to the chain, so it becomes the new head.
				err = testChain.PendingBlockCommit()
				if err != nil {
					return nil, err
				}

				// Ensure our transaction succeeded and, if it did not, attach an execution trace to it and re-run it.
				// The execution trace will be returned so that it can be provided to the user for debugging
				if block.MessageResults[0].Receipt.Status != types.ReceiptStatusSuccessful {
					// Create a call sequence element to represent the failed contract deployment tx
					cse := calls.NewCallSequenceElement(nil, msg, 0, 0)
					cse.ChainReference = &calls.CallSequenceElementChainReference{
						Block:            block,
						TransactionIndex: len(block.Messages) - 1,
					}
					// Revert to genesis and re-run the failed contract deployment tx.
					// We should be able to attach an execution trace; however, if it fails, we provide the ExecutionResult at a minimum.
					err = testChain.RevertToBlockNumber(0)
					if err != nil {
						return nil, fmt.Errorf("failed to reset to genesis block: %v", err)
					} else {
						_, err = calls.ExecuteCallSequenceWithExecutionTracer(testChain, fuzzer.contractDefinitions, []*calls.CallSequenceElement{cse}, true)
						if err != nil {
							return nil, fmt.Errorf("deploying %s returned a failed status: %v", contractName, block.MessageResults[0].ExecutionResult.Err)
						}
					}

					// Return the execution error and the execution trace, if possible.
					return cse.ExecutionTrace, fmt.Errorf("deploying %s returned a failed status: %v", contractName, block.MessageResults[0].ExecutionResult.Err)
				}

				// Record our deployed contract so the next config-specified constructor args can reference this
				// contract by name.
				deployedContractAddr[contractName] = block.MessageResults[0].Receipt.ContractAddress

				// Flag that we found a matching compiled contract definition and deployed it, then exit out of this
				// inner loop to process the next contract to deploy in the outer loop.
				found = true
				break
			}
		}

		// If we did not find a contract corresponding to this item in the deployment order, we throw an error.
		if !found {
			return nil, fmt.Errorf("%v was specified in the target contracts but was not found in the compilation artifacts", contractName)
		}
	}
	return nil, nil
}

// defaultCallSequenceGeneratorConfigFunc is a NewCallSequenceGeneratorConfigFunc which creates a
// CallSequenceGeneratorConfig with a default configuration. Returns the config or an error, if one occurs.
func defaultCallSequenceGeneratorConfigFunc(fuzzer *Fuzzer, valueSet *valuegeneration.ValueSet, randomProvider *rand.Rand) (*CallSequenceGeneratorConfig, error) {
	// Create the value generator and mutator for the worker.
	mutationalGeneratorConfig := &valuegeneration.MutationalValueGeneratorConfig{
		MinMutationRounds:               0,
		MaxMutationRounds:               1,
		GenerateRandomAddressBias:       0.5,
		GenerateRandomIntegerBias:       0.5,
		GenerateRandomStringBias:        0.5,
		GenerateRandomBytesBias:         0.5,
		MutateAddressProbability:        0.1,
		MutateArrayStructureProbability: 0.1,
		MutateBoolProbability:           0.1,
		MutateBytesProbability:          0.1,
		MutateBytesGenerateNewBias:      0.45,
		MutateFixedBytesProbability:     0.1,
		MutateStringProbability:         0.1,
		MutateStringGenerateNewBias:     0.7,
		MutateIntegerProbability:        0.1,
		MutateIntegerGenerateNewBias:    0.5,
		RandomValueGeneratorConfig: &valuegeneration.RandomValueGeneratorConfig{
			GenerateRandomArrayMinSize:  0,
			GenerateRandomArrayMaxSize:  100,
			GenerateRandomBytesMinSize:  0,
			GenerateRandomBytesMaxSize:  100,
			GenerateRandomStringMinSize: 0,
			GenerateRandomStringMaxSize: 100,
		},
	}
	mutationalGenerator := valuegeneration.NewMutationalValueGenerator(mutationalGeneratorConfig, valueSet, randomProvider)

	// Create a sequence generator config which uses the created value generator.
	sequenceGenConfig := &CallSequenceGeneratorConfig{
		NewSequenceProbability:                   0.3,
		RandomUnmodifiedCorpusHeadWeight:         800,
		RandomUnmodifiedCorpusTailWeight:         100,
		RandomUnmodifiedSpliceAtRandomWeight:     200,
		RandomUnmodifiedInterleaveAtRandomWeight: 100,
		RandomMutatedCorpusHeadWeight:            80,
		RandomMutatedCorpusTailWeight:            10,
		RandomMutatedSpliceAtRandomWeight:        20,
		RandomMutatedInterleaveAtRandomWeight:    10,
		ValueGenerator:                           mutationalGenerator,
		ValueMutator:                             mutationalGenerator,
	}
	return sequenceGenConfig, nil
}

// defaultShrinkingValueMutatorFunc is a NewShrinkingValueMutatorFunc which creates value mutator to be used for
// shrinking purposes. Returns the value mutator or an error, if one occurs.
func defaultShrinkingValueMutatorFunc(fuzzer *Fuzzer, valueSet *valuegeneration.ValueSet, randomProvider *rand.Rand) (valuegeneration.ValueMutator, error) {
	// Create the shrinking value mutator for the worker.
	shrinkingValueMutatorConfig := &valuegeneration.ShrinkingValueMutatorConfig{
		ShrinkValueProbability: 0.1,
	}
	shrinkingValueMutator := valuegeneration.NewShrinkingValueMutator(shrinkingValueMutatorConfig, valueSet, randomProvider)
	return shrinkingValueMutator, nil
}

// spawnWorkersLoop is a method which spawns a config-defined amount of FuzzerWorker to carry out the fuzzing campaign.
// This function exits when Fuzzer.ctx is cancelled.
func (f *Fuzzer) spawnWorkersLoop(baseTestChain *chain.TestChain) error {
	// We create our fuzz workers in a loop, using a channel to block when we reach capacity.
	// If we encounter any errors, we stop.
	f.workers = make([]*FuzzerWorker, f.config.Fuzzing.Workers)
	threadReserveChannel := make(chan struct{}, f.config.Fuzzing.Workers)

	// Workers are "reset" when they hit some config-defined limit. They are destroyed and recreated at the same index.
	// For now, we create our available index queue before initializing some providers and entering our main loop.
	type availableWorkerSlot struct {
		index          int
		randomProvider *rand.Rand
	}
	availableWorkerSlotQueue := make([]availableWorkerSlot, f.config.Fuzzing.Workers)
	availableWorkerIndexedLock := sync.Mutex{}
	for i := 0; i < len(availableWorkerSlotQueue); i++ {
		availableWorkerSlotQueue[i] = availableWorkerSlot{
			index:          i,
			randomProvider: randomutils.ForkRandomProvider(f.randomProvider),
		}
	}

	// Define a flag that indicates whether we have not cancelled o
	working := !utils.CheckContextDone(f.ctx)

	// Create workers and start fuzzing.
	var err error
	for err == nil && working {
		// Send an item into our channel to queue up a spot. This will block us if we hit capacity until a worker
		// slot is freed up.
		threadReserveChannel <- struct{}{}

		// Pop a worker index off of our queue
		availableWorkerIndexedLock.Lock()
		workerSlotInfo := availableWorkerSlotQueue[0]
		availableWorkerSlotQueue = availableWorkerSlotQueue[1:]
		availableWorkerIndexedLock.Unlock()

		// Run our goroutine. This should take our queued struct out of the channel once it's done,
		// keeping us at our desired thread capacity. If we encounter an error, we store it and continue
		// processing the cleanup logic to exit gracefully.
		go func(workerSlotInfo availableWorkerSlot) {
			// Create a new worker for this fuzzing.
			worker, workerCreatedErr := newFuzzerWorker(f, workerSlotInfo.index, workerSlotInfo.randomProvider)
			f.workers[workerSlotInfo.index] = worker
			if err == nil && workerCreatedErr != nil {
				err = workerCreatedErr
			}
			if err == nil {
				// Publish an event indicating we created a worker.
				workerCreatedErr = f.Events.WorkerCreated.Publish(FuzzerWorkerCreatedEvent{Worker: worker})
				if err == nil && workerCreatedErr != nil {
					err = workerCreatedErr
				}
			}

			// Run the worker and check if we received a cancelled signal, or we encountered an error.
			if err == nil {
				ctxCancelled, workerErr := worker.run(baseTestChain)
				if workerErr != nil {
					err = workerErr
				}

				// If we received a cancelled signal, signal our exit from the working loop.
				if working && ctxCancelled {
					working = false
				}
			}

			// Free our worker id before unblocking our channel, as a free one will be expected.
			availableWorkerIndexedLock.Lock()
			availableWorkerSlotQueue = append(availableWorkerSlotQueue, workerSlotInfo)
			availableWorkerIndexedLock.Unlock()

			// Publish an event indicating we destroyed a worker.
			workerDestroyedErr := f.Events.WorkerDestroyed.Publish(FuzzerWorkerDestroyedEvent{Worker: worker})
			if err == nil && workerDestroyedErr != nil {
				err = workerDestroyedErr
			}

			// Unblock our channel by freeing our capacity of another item, making way for another worker.
			<-threadReserveChannel
		}(workerSlotInfo)
	}

	// Explicitly call cancel on our context to ensure all threads exit if we encountered an error.
	if f.ctxCancelFunc != nil {
		f.ctxCancelFunc()
	}

	// Wait for every worker to be freed, so we don't have a race condition when reporting the order
	// of events to our test provider.
	for {
		// Obtain the count of free workers.
		availableWorkerIndexedLock.Lock()
		freeWorkers := len(availableWorkerSlotQueue)
		availableWorkerIndexedLock.Unlock()

		// We keep waiting until every worker is free
		if freeWorkers == len(f.workers) {
			break
		} else {
			time.Sleep(50 * time.Millisecond)
		}
	}
	return err
}

// Start begins a fuzzing operation on the provided project configuration. This operation will not return until an error
// is encountered or the fuzzing operation has completed. Its execution can be cancelled using the Stop method.
// Returns an error if one is encountered.
func (f *Fuzzer) Start() error {
	// Define our variable to catch errors
	var err error

	// While we're fuzzing, we'll want to have an initialized random provider.
	f.randomProvider = rand.New(rand.NewSource(time.Now().UnixNano()))

	// Create our running context (allows us to cancel across threads)
	f.ctx, f.ctxCancelFunc = context.WithCancel(context.Background())

	// If we set a timeout, create the timeout context now, as we're about to begin fuzzing.
	if f.config.Fuzzing.Timeout > 0 {
		f.logger.Info("Running with a timeout of ", colors.Bold, f.config.Fuzzing.Timeout, " seconds")
		f.ctx, f.ctxCancelFunc = context.WithTimeout(f.ctx, time.Duration(f.config.Fuzzing.Timeout)*time.Second)
	}

	// Set up the corpus
	f.logger.Info("Initializing corpus")
	f.corpus, err = corpus.NewCorpus(f.config.Fuzzing.CorpusDirectory)
	if err != nil {
		f.logger.Error("Failed to create the corpus", err)
		return err
	}

	// Initialize our metrics and valueGenerator.
	f.metrics = newFuzzerMetrics(f.config.Fuzzing.Workers)

	// Initialize our test cases and providers
	f.testCasesLock.Lock()
	f.testCases = make([]TestCase, 0)
	f.testCasesFinished = make(map[string]TestCase)
	f.testCasesLock.Unlock()

	// Create our test chain
	baseTestChain, err := f.createTestChain()
	if err != nil {
		f.logger.Error("Failed to create the test chain", err)
		return err
	}

	// Set it up with our deployment/setup strategy defined by the fuzzer.
	f.logger.Info("Setting up test chain")
	trace, err := f.Hooks.ChainSetupFunc(f, baseTestChain)
	if err != nil {
		if trace != nil {
			f.logger.Error("Failed to initialize the test chain", err, errors.New(trace.Log().ColorString()))
		} else {
			f.logger.Error("Failed to initialize the test chain", err)
		}
		return err
	}
	f.logger.Info("Finished setting up test chain")

	// Initialize our coverage maps by measuring the coverage we get from the corpus.
	var corpusActiveSequences, corpusTotalSequences int
	if totalCallSequences, testResults := f.corpus.CallSequenceEntryCount(); totalCallSequences > 0 || testResults > 0 {
		f.logger.Info("Running call sequences in the corpus")
	}
	startTime := time.Now()
	corpusActiveSequences, corpusTotalSequences, err = f.corpus.Initialize(baseTestChain, f.contractDefinitions)
	if corpusTotalSequences > 0 {
		f.logger.Info("Finished running call sequences in the corpus in ", time.Since(startTime).Round(time.Second))
	}
	if err != nil {
		f.logger.Error("Failed to initialize the corpus", err)
		return err
	}

	// Log corpus health statistics, if we have any existing sequences.
	if corpusTotalSequences > 0 {
		f.logger.Info(
			colors.Bold, "corpus: ", colors.Reset,
			"health: ", colors.Bold, int(float32(corpusActiveSequences)/float32(corpusTotalSequences)*100.0), "%", colors.Reset, ", ",
			"sequences: ", colors.Bold, corpusTotalSequences, " (", corpusActiveSequences, " valid, ", corpusTotalSequences-corpusActiveSequences, " invalid)", colors.Reset,
		)
	}

	// Log the start of our fuzzing campaign.
	f.logger.Info("Fuzzing with ", colors.Bold, f.config.Fuzzing.Workers, colors.Reset, " workers")

	// Start our printing loop now that we're about to begin fuzzing.
	go f.printMetricsLoop()

	// Publish a fuzzer starting event.
	err = f.Events.FuzzerStarting.Publish(FuzzerStartingEvent{Fuzzer: f})
	if err != nil {
		f.logger.Error("FuzzerStarting event subscriber returned an error", err)
		return err
	}

	// If StopOnNoTests is true and there are no test cases, then throw an error
	if f.config.Fuzzing.Testing.StopOnNoTests && len(f.testCases) == 0 {
		err = fmt.Errorf("no assertion, property, optimization, or custom tests were found to fuzz")
		if !f.config.Fuzzing.Testing.AssertionTesting.TestViewMethods {
			err = fmt.Errorf("no assertion, property, optimization, or custom tests were found to fuzz and testing view methods is disabled")
		}
		f.logger.Error("Failed to start fuzzer", err)
		return err
	}

	// Run the main worker loop
	err = f.spawnWorkersLoop(baseTestChain)
	if err != nil {
		f.logger.Error("Encountered an error in the main fuzzing loop", err)
	}

	// NOTE: After this point, we capture errors but do not return immediately, as we want to exit gracefully.

	// If we have coverage enabled and a corpus directory set, write the corpus. We do this even if we had a
	// previous error, as we don't want to lose corpus entries.
	if f.config.Fuzzing.CoverageEnabled {
		corpusFlushErr := f.corpus.Flush()
		if err == nil && corpusFlushErr != nil {
			err = corpusFlushErr
			f.logger.Info("Failed to flush the corpus", err)
		}
	}

	// Publish a fuzzer stopping event.
	fuzzerStoppingErr := f.Events.FuzzerStopping.Publish(FuzzerStoppingEvent{Fuzzer: f, err: err})
	if err == nil && fuzzerStoppingErr != nil {
		err = fuzzerStoppingErr
		f.logger.Error("FuzzerStopping event subscriber returned an error", err)
	}

	// Print our results on exit.
	f.printExitingResults()

	// Finally, generate our coverage report if we have set a valid corpus directory.
	if err == nil && len(f.config.Fuzzing.CoverageFormats) > 0 {
		// Write to the default directory if we have no corpus directory set.
		coverageReportDir := filepath.Join("crytic-export", "coverage")
		if f.config.Fuzzing.CorpusDirectory != "" {
			coverageReportDir = filepath.Join(f.config.Fuzzing.CorpusDirectory, "coverage")
		}
		sourceAnalysis, err := coverage.AnalyzeSourceCoverage(f.compilations, f.corpus.CoverageMaps())

		if err != nil {
			f.logger.Error("Failed to analyze source coverage", err)
		} else {
			var path string
			for _, reportType := range f.config.Fuzzing.CoverageFormats {
				switch reportType {
				case "html":
					path, err = coverage.WriteHTMLReport(sourceAnalysis, coverageReportDir)
				case "lcov":
					path, err = coverage.WriteLCOVReport(sourceAnalysis, coverageReportDir)
				default:
					err = fmt.Errorf("unsupported coverage report type: %s", reportType)
				}
				if err != nil {
					f.logger.Error(fmt.Sprintf("Failed to generate %s coverage report", reportType), err)
				} else {
					f.logger.Info(fmt.Sprintf("%s report(s) saved to: %s", reportType, path), colors.Bold, colors.Reset)
				}
			}
		}
	}

	// Return any encountered error.
	return err
}

// Stop stops a running operation invoked by the Start method. This method may return before complete operation teardown
// occurs.
func (f *Fuzzer) Stop() {
	// Call the cancel function on our running context to stop all working goroutines
	if f.ctxCancelFunc != nil {
		f.ctxCancelFunc()
	}
}

// printMetricsLoop prints metrics to the console in a loop until ctx signals a stopped operation.
func (f *Fuzzer) printMetricsLoop() {
	// Define our start time
	startTime := time.Now()

	// Define cached variables for our metrics to calculate deltas.
	lastCallsTested := big.NewInt(0)
	lastSequencesTested := big.NewInt(0)
	lastWorkerStartupCount := big.NewInt(0)
	lastGasUsed := big.NewInt(0)

	lastPrintedTime := time.Time{}
	for !utils.CheckContextDone(f.ctx) {
		// Obtain our metrics
		callsTested := f.metrics.CallsTested()
		sequencesTested := f.metrics.SequencesTested()
		gasUsed := f.metrics.GasUsed()
		failedSequences := f.metrics.FailedSequences()
		workerStartupCount := f.metrics.WorkerStartupCount()
		workersShrinking := f.metrics.WorkersShrinkingCount()

		// Calculate time elapsed since the last update
		secondsSinceLastUpdate := time.Since(lastPrintedTime).Seconds()

		// Obtain memory usage stats
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		memoryUsedMB := memStats.Alloc / 1024 / 1024
		memoryTotalMB := memStats.Sys / 1024 / 1024

		// Print a metrics update
		logBuffer := logging.NewLogBuffer()
		logBuffer.Append(colors.Bold, "fuzz: ", colors.Reset)
		logBuffer.Append("elapsed: ", colors.Bold, time.Since(startTime).Round(time.Second).String(), colors.Reset)
		logBuffer.Append(", calls: ", colors.Bold, fmt.Sprintf("%d (%d/sec)", callsTested, uint64(float64(new(big.Int).Sub(callsTested, lastCallsTested).Uint64())/secondsSinceLastUpdate)), colors.Reset)
		logBuffer.Append(", seq/s: ", colors.Bold, fmt.Sprintf("%d", uint64(float64(new(big.Int).Sub(sequencesTested, lastSequencesTested).Uint64())/secondsSinceLastUpdate)), colors.Reset)
		logBuffer.Append(", coverage: ", colors.Bold, fmt.Sprintf("%d", f.corpus.CoverageMaps().UniquePCs()), colors.Reset)
		logBuffer.Append(", corpus: ", colors.Bold, fmt.Sprintf("%d", f.corpus.ActiveMutableSequenceCount()), colors.Reset)
		logBuffer.Append(", failures: ", colors.Bold, fmt.Sprintf("%d/%d", failedSequences, sequencesTested), colors.Reset)
		logBuffer.Append(", gas/s: ", colors.Bold, fmt.Sprintf("%d", uint64(float64(new(big.Int).Sub(gasUsed, lastGasUsed).Uint64())/secondsSinceLastUpdate)), colors.Reset)
		if f.logger.Level() <= zerolog.DebugLevel {
			logBuffer.Append(", shrinking: ", colors.Bold, fmt.Sprintf("%v", workersShrinking), colors.Reset)
			logBuffer.Append(", mem: ", colors.Bold, fmt.Sprintf("%v/%v MB", memoryUsedMB, memoryTotalMB), colors.Reset)
			logBuffer.Append(", resets/s: ", colors.Bold, fmt.Sprintf("%d", uint64(float64(new(big.Int).Sub(workerStartupCount, lastWorkerStartupCount).Uint64())/secondsSinceLastUpdate)), colors.Reset)
		}
		f.logger.Info(logBuffer.Elements()...)

		// Update our delta tracking metrics
		lastPrintedTime = time.Now()
		lastCallsTested = callsTested
		lastSequencesTested = sequencesTested
		lastGasUsed = gasUsed
		lastWorkerStartupCount = workerStartupCount

		// If we reached our transaction threshold, halt
		testLimit := f.config.Fuzzing.TestLimit
		if testLimit > 0 && (!callsTested.IsUint64() || callsTested.Uint64() >= testLimit) {
			f.logger.Info("Transaction test limit reached, halting now...")
			f.Stop()
			break
		}

		// Sleep some time between print iterations
		time.Sleep(time.Second * 3)
	}
}

// printExitingResults prints the TestCase results prior to the fuzzer exiting.
func (f *Fuzzer) printExitingResults() {
	// Define the order our test cases should be sorted by when considering status.
	testCaseDisplayOrder := map[TestCaseStatus]int{
		TestCaseStatusNotStarted: 0,
		TestCaseStatusPassed:     1,
		TestCaseStatusFailed:     2,
		TestCaseStatusRunning:    3,
	}

	// Sort the test cases by status and then ID.
	sort.Slice(f.testCases, func(i int, j int) bool {
		// Sort by order first
		iStatusOrder := testCaseDisplayOrder[f.testCases[i].Status()]
		jStatusOrder := testCaseDisplayOrder[f.testCases[j].Status()]
		if iStatusOrder != jStatusOrder {
			return iStatusOrder < jStatusOrder
		}

		// Then we sort by ID.
		return strings.Compare(f.testCases[i].ID(), f.testCases[j].ID()) <= 0
	})

	// Define variables to track our final test count.
	var (
		testCountPassed int
		testCountFailed int
	)

	// Print the results of each individual test case.
	f.logger.Info("Fuzzer stopped, test results follow below ...")
	for _, testCase := range f.testCases {
		f.logger.Info(testCase.LogMessage().ColorString())

		// Tally our pass/fail count.
		if testCase.Status() == TestCaseStatusPassed {
			testCountPassed++
		} else if testCase.Status() == TestCaseStatusFailed {
			testCountFailed++
		}
	}

	// Print our final tally of test statuses.
	f.logger.Info("Test summary: ", colors.GreenBold, testCountPassed, colors.Reset, " test(s) passed, ", colors.RedBold, testCountFailed, colors.Reset, " test(s) failed")
}
