package fuzzing

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/trailofbits/medusa/chain"
	compilationTypes "github.com/trailofbits/medusa/compilation/types"
	"github.com/trailofbits/medusa/events"
	"github.com/trailofbits/medusa/fuzzing/config"
	corpusTypes "github.com/trailofbits/medusa/fuzzing/corpus"
	"github.com/trailofbits/medusa/fuzzing/coverage"
	fuzzerTypes "github.com/trailofbits/medusa/fuzzing/types"
	"github.com/trailofbits/medusa/fuzzing/value_generation"
	"github.com/trailofbits/medusa/utils"
	"golang.org/x/exp/slices"
	"math/big"
	"sync"
	"time"
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
	// contractDefinitions defines targets to be fuzzed once their deployment is detected.
	contractDefinitions []fuzzerTypes.Contract

	// NewValueGeneratorFunc describes the function to use to set up a new value generator per worker.
	NewValueGeneratorFunc NewValueGeneratorFunc
	// ChainSetupFunc describes the function to use to set up a new test chain's initial state prior to fuzzing.
	ChainSetupFunc TestChainSetupFunc

	// CallSequenceTestFunctions describes a list of functions to be called upon by a FuzzerWorker after every call
	// in a call sequence.
	CallSequenceTestFunctions []CallSequenceTestFunc

	// workers represents the work threads created by this Fuzzer when Start invokes a fuzz operation.
	workers []*FuzzerWorker
	// metrics represents the metrics for the fuzzing campaign.
	metrics *FuzzerMetrics
	// corpus stores a list of transaction sequences that can be used for coverage-guided fuzzing
	corpus *corpusTypes.Corpus

	// baseValueSet represents a value_generation.ValueSet containing input values for our fuzz tests.
	baseValueSet *value_generation.ValueSet

	// testCases contains every TestCase registered with the Fuzzer.
	testCases []TestCase
	// testCasesLock provides thread-synchronization to avoid race conditions when accessing or updating test cases.
	testCasesLock sync.Mutex
	// testCasesFinished describes test cases already reported as having been finalized.
	testCasesFinished map[string]TestCase

	// OnStartingEventEmitter emits events when the Fuzzer initialized state and is ready to about to begin the main
	// execution loop for the fuzzing campaign.
	OnStartingEventEmitter events.EventEmitter[OnFuzzerStarting]
	// OnStoppingEventEmitter emits events when the Fuzzer is exiting its main fuzzing loop.
	OnStoppingEventEmitter events.EventEmitter[OnFuzzerStopping]
	// OnWorkerCreatedEventEmitter emits events when the Fuzzer creates a new FuzzerWorker during the fuzzing campaign.
	OnWorkerCreatedEventEmitter events.EventEmitter[OnWorkerCreated]
	// OnWorkerDestroyedEventEmitter emits events when the Fuzzer destroys an existing FuzzerWorker during the fuzzing
	// campaign. This can occur even if a fuzzing campaign is not stopping, if a worker has reached resource limits.
	OnWorkerDestroyedEventEmitter events.EventEmitter[OnWorkerDestroyed]
}

// NewValueGeneratorFunc defines a method which is called to create a value_generation.ValueGenerator for a worker
// when it is created. It takes the current fuzzer as an argument for context, and is expected to return a generator,
// or an error if one is encountered.
type NewValueGeneratorFunc func(fuzzer *Fuzzer) (value_generation.ValueGenerator, error)

// TestChainSetupFunc describes a function which sets up a test chain's initial state prior to fuzzing.
type TestChainSetupFunc func(fuzzer *Fuzzer, testChain *chain.TestChain) error

// NewFuzzer returns an instance of a new Fuzzer provided a project configuration, or an error if one is encountered
// while initializing the code.
func NewFuzzer(config config.ProjectConfig) (*Fuzzer, error) {
	// TODO: Validate config variables (e.g. worker count > 0)
	// Parse the senders addresses from our account config.
	senders, err := utils.HexStringsToAddresses(config.Fuzzing.SenderAddresses)
	if err != nil {
		return nil, err
	}

	// Parse the deployer address from our account config
	deployer, err := utils.HexStringToAddress(config.Fuzzing.DeployerAddress)
	if err != nil {
		return nil, err
	}

	// Create and return our fuzzing instance.
	fuzzer := &Fuzzer{
		config:                    config,
		senders:                   senders,
		deployer:                  deployer,
		baseValueSet:              value_generation.NewValueSet(),
		contractDefinitions:       make([]fuzzerTypes.Contract, 0),
		testCases:                 make([]TestCase, 0),
		testCasesFinished:         make(map[string]TestCase),
		NewValueGeneratorFunc:     createDefaultValueGenerator,
		ChainSetupFunc:            deploymentStrategyCompilationConfig,
		CallSequenceTestFunctions: make([]CallSequenceTestFunc, 0),
	}

	// If we have a compilation config
	if fuzzer.config.Compilation != nil {
		// Compile the targets specified in the compilation config
		fmt.Printf("Compiling targets (platform '%s') ...\n", fuzzer.config.Compilation.Platform)
		compilations, compilationOutput, err := (*fuzzer.config.Compilation).Compile()
		if err != nil {
			return nil, err
		}
		fmt.Printf(compilationOutput)

		// Add our compilation targets
		fuzzer.AddCompilationTargets(compilations)
	}

	// Register any default providers if specified.
	if fuzzer.config.Fuzzing.Testing.PropertyTesting.Enabled {
		attachPropertyTestCaseProvider(fuzzer)
	}
	if fuzzer.config.Fuzzing.Testing.AssertionTesting.Enabled {
		// TODO: Add assertion test provider.
	}
	return fuzzer, nil
}

// ContractDefinitions exposes the contract definitions registered with the Fuzzer.
func (f *Fuzzer) ContractDefinitions() []fuzzerTypes.Contract {
	return slices.Clone(f.contractDefinitions)
}

// Config exposes the underlying project configuration provided to the Fuzzer.
func (f *Fuzzer) Config() config.ProjectConfig {
	return f.config
}

// BaseValueSet exposes the underlying value set provided to the Fuzzer value generators to aid in generation
// (e.g. for use in mutation operations).
func (f *Fuzzer) BaseValueSet() *value_generation.ValueSet {
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
		fmt.Printf("\n[%s] %s\n%s\n\n", testCase.Status(), testCase.Name(), testCase.Message())
	}

	// If the config specifies, we stop after the first failed test reported.
	if testCase.Status() == TestCaseStatusFailed && f.config.Fuzzing.Testing.StopOnFailedTest {
		f.Stop()
	}
}

// AddCompilationTargets takes a compilation and updates the Fuzzer state with additional Fuzzer.ContractDefinitions
// definitions and Fuzzer.BaseValueSet values.
func (f *Fuzzer) AddCompilationTargets(compilations []compilationTypes.Compilation) {
	// Loop for each contract in each compilation and deploy it to the test node.
	for _, comp := range compilations {
		for _, source := range comp.Sources {
			// Seed our base value set from every source's AST
			f.baseValueSet.SeedFromAst(source.Ast)

			// Loop for every contract and register it in our contract definitions
			for contractName := range source.Contracts {
				contract := source.Contracts[contractName]
				contractDefinition := fuzzerTypes.NewContract(contractName, &contract)
				f.contractDefinitions = append(f.contractDefinitions, *contractDefinition)
			}
		}
	}
}

// createTestChain creates a test chain with the account balance allocations specified by the config.
func (f *Fuzzer) createTestChain() (*chain.TestChain, error) {
	// Create our genesis allocations.
	// NOTE: Sharing GenesisAlloc between nodes will result in some accounts not being funded for some reason.
	genesisAlloc := make(core.GenesisAlloc)

	// Fund all of our sender addresses in the genesis block
	initBalance := new(big.Int).Div(abi.MaxInt256, big.NewInt(2)) // TODO: make this configurable
	for _, sender := range f.senders {
		genesisAlloc[sender] = core.GenesisAccount{
			Balance: initBalance,
		}
	}

	// Fund our deployer address in the genesis block
	genesisAlloc[f.deployer] = core.GenesisAccount{
		Balance: initBalance,
	}

	// Create our test chain with our basic allocations.
	testChain, err := chain.NewTestChain(genesisAlloc)
	return testChain, err
}

// initializeCoverageMaps initializes our coverage maps by tracing initial deployments in the test chain and replaying
// all call sequences stored in the corpus.
func (f *Fuzzer) initializeCoverageMaps(baseTestChain *chain.TestChain) error {
	// Create a coverage tracer
	coverageTracer := coverage.NewCoverageTracer()

	// Clone our test chain with our coverage tracer.
	testChain, err := baseTestChain.Clone(coverageTracer)
	if err != nil {
		return fmt.Errorf("failed to initialize coverage maps, base test chain cloning encountered error: %v", err)
	}

	// Add our coverage recorded during chain setup to our coverage maps.
	_, err = f.corpus.CoverageMaps().Update(coverageTracer.CoverageMaps())
	if err != nil {
		return fmt.Errorf("failed to initialize coverage maps, base test chain coverage update encountered error: %v", err)
	}

	// Next we measure coverage for every corpus call sequence.
	corpusCallSequences := f.corpus.CallSequences()

	// Cache current HeadBlockNumber so that you can reset back to it after every sequence
	baseBlockNumber := testChain.HeadBlockNumber()

	for _, sequence := range corpusCallSequences {
		// Reset our tracer's coverage maps.
		coverageTracer.Reset()

		// Loop through each block in the sequence
		// Note that `block`'s blockNumber may be less/greater/equal to `testChain.HeadBlockNumber()`
		// To maintain chain semantics we have to find the difference in block number (and timestamp)
		// and may need to rebase blocks.
		// Note: original corpus block number preservation is prioritized over block gaps between corpus blocks.
		for i, block := range sequence.Blocks {
			// If the next corpus block is ahead of the chain's head, that will be the `nextBlockNumber`
			// Same reasoning for `nextTs`
			rebasedBlockNumber := block.Header.Number.Uint64()
			rebasedTimestamp := block.Header.Timestamp

			// Get difference in head and corpus block numbers
			blockNumberDiff := int(block.Header.Number.Uint64()) - int(testChain.HeadBlockNumber())

			// If the chain head is ahead, or at the same level, as the corpus block (blockNumberDiff <= 0)
			// The next block number should be the head's block number plus the gap between the current corpus block
			// and the previous one
			if blockNumberDiff <= 0 {
				// At minimum, the timestamp and block must be shifted 1 block past the HEAD (in the diff==0 case).
				blockGap := 1
				timestampGap := 1

				// If this isn't the first block we're trying to add, we evaluate the gap between blocks to ensure
				// we maintain it.
				if i > 0 {
					blockGap = int(block.Header.Number.Uint64()) - int(sequence.Blocks[i-1].Header.Number.Uint64())
					timestampGap = int(block.Header.Timestamp) - int(sequence.Blocks[i-1].Header.Timestamp)
				}

				// The next block is head block number plus the gap. Same applies to timestamp.
				rebasedBlockNumber = testChain.HeadBlockNumber() + uint64(blockGap)
				rebasedTimestamp = testChain.Head().Header().Time + uint64(timestampGap)
			}

			// Mine the block with the new block number + timestamp, and messages from the corpus block
			_, err = testChain.MineBlockWithParameters(rebasedBlockNumber, rebasedTimestamp, utils.SliceValuesToPointers(block.Transactions)...)
			if err != nil {
				return fmt.Errorf("failed to mine block, mining encountered an error: %v", err)
			}
		}

		// Add our coverage recorded during call sequence execution to our coverage maps.
		_, err = f.corpus.CoverageMaps().Update(coverageTracer.CoverageMaps())
		if err != nil {
			return fmt.Errorf("failed to initialize coverage maps, call sequence encountered error: %v", err)
		}

		// Revert chain state to our starting point to test the next sequence.
		err = testChain.RevertToBlockNumber(baseBlockNumber)
		if err != nil {
			return fmt.Errorf("failed to reset the chain while seeding coverage: %v\n", err)
		}
	}
	return nil
}

// deploymentStrategyCompilationConfig is a TestChainSetupFunc which sets up the base test chain state by deploying
// all contracts. First, the contracts in the DeploymentOrder config variable are deployed. Then, any remaining
// contracts are deployed.
func deploymentStrategyCompilationConfig(fuzzer *Fuzzer, testChain *chain.TestChain) error {
	// Loop for all contracts to deploy
	for _, contractName := range fuzzer.config.Fuzzing.DeploymentOrder {
		// Look for a contract in our compiled contract definitions that matches this one
		found := false
		for _, contract := range fuzzer.contractDefinitions {
			// If we found a contract definition that matches this definition by name, try to deploy it
			if contract.Name() == contractName {
				// If the contract has no constructor args, deploy it. Only these contracts are supported for now.
				// TODO: We can add logic for deploying contracts with constructor arguments here.
				if len(contract.CompiledContract().Abi.Constructor.Inputs) == 0 {
					// Deploy the contract using our deployer address.
					_, _, err := testChain.DeployContract(contract.CompiledContract(), fuzzer.deployer)
					if err != nil {
						return err
					}
				}

				// Set our found flag to true.
				found = true
				break
			}
		}

		// If we did not find a contract corresponding to this item in the deployment order, we throw an error.
		if !found {
			return fmt.Errorf("DeploymentOrder specified a contract name which was not found in the compilation: %v\n", contractName)
		}
	}
	return nil
}

func createDefaultValueGenerator(fuzzer *Fuzzer) (value_generation.ValueGenerator, error) {
	valueGenConfig := &value_generation.MutatingValueGeneratorConfig{
		MinMutationRounds: 0,
		MaxMutationRounds: 3,
		RandomIntegerBias: 0.2,
		RandomStringBias:  0.2,
		RandomBytesBias:   0.2,
		RandomValueGeneratorConfig: &value_generation.RandomValueGeneratorConfig{
			RandomArrayMinSize:  0,
			RandomArrayMaxSize:  100,
			RandomBytesMinSize:  0,
			RandomBytesMaxSize:  100,
			RandomStringMinSize: 0,
			RandomStringMaxSize: 100,
		},
		ValueSet: fuzzer.baseValueSet.Clone(),
	}
	valueGenerator := value_generation.NewMutatingValueGenerator(valueGenConfig)
	return valueGenerator, nil
}

// Start begins a fuzzing operation on the provided project configuration. This operation will not return until an error
// is encountered or the fuzzing operation has completed. Its execution can be cancelled using the Stop method.
// Returns an error if one is encountered.
func (f *Fuzzer) Start() error {
	// Define our variable to catch errors
	var err error

	// Create our running context (allows us to cancel across threads)
	f.ctx, f.ctxCancelFunc = context.WithCancel(context.Background())

	// If we set a timeout, create the timeout context now, as we're about to begin fuzzing.
	if f.config.Fuzzing.Timeout > 0 {
		fmt.Printf("Running with timeout of %d seconds\n", f.config.Fuzzing.Timeout)
		f.ctx, f.ctxCancelFunc = context.WithTimeout(f.ctx, time.Duration(f.config.Fuzzing.Timeout)*time.Second)
	}

	// Set up the corpus
	f.corpus, err = corpusTypes.NewCorpus(f.config.Fuzzing.CorpusDirectory)
	if err != nil {
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
		return err
	}

	// Set it up with our deployment/setup strategy defined by the fuzzer.
	err = f.ChainSetupFunc(f, baseTestChain)
	if err != nil {
		return err
	}

	// Initialize our coverage maps
	err = f.initializeCoverageMaps(baseTestChain)
	if err != nil {
		return err
	}

	// We create a test node for each thread we intend to create. Fuzzer workers can stop if they hit some resource
	// limit such as a memory limit, at which point we'll recreate them in our loop, putting them into the same index.
	// For now, we create our available index queue before initializing some providers and entering our main loop.
	availableWorkerIndexes := make([]int, f.config.Fuzzing.Workers)
	availableWorkerIndexedLock := sync.Mutex{}
	for i := 0; i < len(availableWorkerIndexes); i++ {
		availableWorkerIndexes[i] = i
	}

	// Start our printing loop now that we're about to begin fuzzing.
	go f.runMetricsPrintLoop()

	// Finally, we create our fuzz workers in a loop, using a channel to block when we reach capacity.
	// If we encounter any errors, we stop.
	f.workers = make([]*FuzzerWorker, f.config.Fuzzing.Workers)
	threadReserveChannel := make(chan struct{}, f.config.Fuzzing.Workers)

	// Publish a fuzzer starting event.
	err = f.OnStartingEventEmitter.Publish(OnFuzzerStarting{Fuzzer: f})

	// If our context is cancelled, perform cleanup and exit instead of entering the fuzzing loop.
	working := !utils.CheckContextDone(f.ctx)

	// Log that we are about to create the workers and start fuzzing
	fmt.Printf("Creating %d workers ...\n", f.config.Fuzzing.Workers)
	for err == nil && working {
		// Send an item into our channel to queue up a spot. This will block us if we hit capacity until a worker
		// slot is freed up.
		threadReserveChannel <- struct{}{}

		// Pop a worker index off of our queue
		availableWorkerIndexedLock.Lock()
		workerIndex := availableWorkerIndexes[0]
		availableWorkerIndexes = availableWorkerIndexes[1:]
		availableWorkerIndexedLock.Unlock()

		// Run our goroutine. This should take our queued struct out of the channel once it's done,
		// keeping us at our desired thread capacity. If we encounter an error, we store it and continue
		// processing the cleanup logic to exit gracefully.
		go func(workerIndex int) {
			// Create a new worker for this fuzzing.
			worker, workerCreatedErr := newFuzzerWorker(f, workerIndex)
			f.workers[workerIndex] = worker
			if err == nil && workerCreatedErr != nil {
				err = workerCreatedErr
			}
			if err == nil {
				// Publish an event indicating we created a worker.
				workerCreatedErr = f.OnWorkerCreatedEventEmitter.Publish(OnWorkerCreated{Worker: worker})
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
			availableWorkerIndexes = append(availableWorkerIndexes, workerIndex)
			availableWorkerIndexedLock.Unlock()

			// Publish an event indicating we destroyed a worker.
			workerDestroyedErr := f.OnWorkerDestroyedEventEmitter.Publish(OnWorkerDestroyed{Worker: worker})
			if err == nil && workerDestroyedErr != nil {
				err = workerDestroyedErr
			}

			// Unblock our channel by freeing our capacity of another item, making way for another worker.
			<-threadReserveChannel
		}(workerIndex)
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
		freeWorkers := len(availableWorkerIndexes)
		availableWorkerIndexedLock.Unlock()

		// We keep waiting until every worker is free
		if freeWorkers == len(f.workers) {
			break
		} else {
			time.Sleep(50 * time.Millisecond)
		}
	}

	// If we have coverage enabled and a corpus directory set, write the corpus. We do this even if we had a
	// previous error, as we don't want to lose corpus entries.
	if f.config.Fuzzing.CoverageEnabled {
		corpusFlushErr := f.corpus.Flush()
		if err == nil {
			err = corpusFlushErr
		}
	}

	// Now that we've gracefully ensured all goroutines have ceased, we can immediately return any errors.
	if err != nil {
		return err
	}

	// Publish a fuzzer stopping event.
	fuzzerStoppingErr := f.OnStoppingEventEmitter.Publish(OnFuzzerStopping{Fuzzer: f})
	if err == nil && fuzzerStoppingErr != nil {
		err = fuzzerStoppingErr
	}

	// Print our test case results
	fmt.Printf("\n")
	fmt.Printf("Fuzzer stopped, test results follow below ...\n")
	for _, testCase := range f.testCases {
		if testCase.Status() == TestCaseStatusFailed {
			fmt.Printf("[%s] %s\n%s\n\n", testCase.Status(), testCase.Name(), testCase.Message())
		} else {
			fmt.Printf("[%s] %s\n", testCase.Status(), testCase.Name())
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

// runMetricsPrintLoop prints metrics to the console in a loop until ctx signals a stopped operation.
func (f *Fuzzer) runMetricsPrintLoop() {
	// Define our start time
	startTime := time.Now()

	// Define cached variables for our metrics to calculate deltas.
	var lastTransactionsTested, lastSequencesTested, lastWorkerStartupCount uint64
	lastPrintedTime := time.Time{}
	for !utils.CheckContextDone(f.ctx) {
		// Obtain our metrics
		transactionsTested := f.metrics.TransactionsTested()
		sequencesTested := f.metrics.SequencesTested()
		workerStartupCount := f.metrics.WorkerStartupCount()

		// Calculate time elapsed since the last update
		secondsSinceLastUpdate := time.Now().Sub(lastPrintedTime).Seconds()

		// Print a metrics update
		fmt.Printf(
			"fuzz: elapsed: %s, call: %d (%d/sec), seq/s: %d, hitmemlimit: %d/s\n",
			time.Now().Sub(startTime).Round(time.Second),
			transactionsTested,
			uint64(float64(transactionsTested-lastTransactionsTested)/secondsSinceLastUpdate),
			uint64(float64(sequencesTested-lastSequencesTested)/secondsSinceLastUpdate),
			uint64(float64(workerStartupCount-lastWorkerStartupCount)/secondsSinceLastUpdate),
		)

		// Update our delta tracking metrics
		lastPrintedTime = time.Now()
		lastTransactionsTested = transactionsTested
		lastSequencesTested = sequencesTested
		lastWorkerStartupCount = workerStartupCount

		// If we reached our transaction threshold, halt
		testLimit := f.config.Fuzzing.TestLimit
		if testLimit > 0 && transactionsTested >= testLimit {
			fmt.Printf("transaction test limit reached, halting now ...\n")
			f.Stop()
			break
		}

		// Sleep for a second
		time.Sleep(time.Second * 3)
	}
}
