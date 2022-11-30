package fuzzing

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/trailofbits/medusa/chain"
	compilationTypes "github.com/trailofbits/medusa/compilation/types"
	"github.com/trailofbits/medusa/fuzzing/config"
	corpusTypes "github.com/trailofbits/medusa/fuzzing/corpus"
	"github.com/trailofbits/medusa/fuzzing/coverage"
	fuzzerTypes "github.com/trailofbits/medusa/fuzzing/types"
	"github.com/trailofbits/medusa/fuzzing/valuegeneration"
	"github.com/trailofbits/medusa/utils"
	"golang.org/x/exp/slices"
	"math/big"
	"strings"
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
	// baseValueSet represents a valuegeneration.ValueSet containing input values for our fuzz tests.
	baseValueSet *valuegeneration.ValueSet

	// workers represents the work threads created by this Fuzzer when Start invokes a fuzz operation.
	workers []*FuzzerWorker
	// metrics represents the metrics for the fuzzing campaign.
	metrics *FuzzerMetrics
	// corpus stores a list of transaction sequences that can be used for coverage-guided fuzzing
	corpus *corpusTypes.Corpus
	// coverageMaps describes the total code coverage known to be achieved across the fuzzing campaign.
	coverageMaps *coverage.CoverageMaps

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
}

// NewFuzzer returns an instance of a new Fuzzer provided a project configuration, or an error if one is encountered
// while initializing the code.
func NewFuzzer(config config.ProjectConfig) (*Fuzzer, error) {
	// Validate our provided config
	err := config.Validate()
	if err != nil {
		return nil, err
	}

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
		config:              config,
		senders:             senders,
		deployer:            deployer,
		baseValueSet:        valuegeneration.NewValueSet(),
		contractDefinitions: make([]fuzzerTypes.Contract, 0),
		testCases:           make([]TestCase, 0),
		testCasesFinished:   make(map[string]TestCase),
		Hooks: FuzzerHooks{
			NewValueGeneratorFunc: defaultNewValueGeneratorFunc,
			ChainSetupFunc:        chainSetupFromCompilations,
			CallSequenceTestFuncs: make([]CallSequenceTestFunc, 0),
		},
		coverageMaps: nil,
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
		attachAssertionTestCaseProvider(fuzzer)
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
	// NOTE: Sharing GenesisAlloc between chains will result in some accounts not being funded for some reason.
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

	// Set our block gas limit
	testChain.BlockGasLimit = f.config.Fuzzing.BlockGasLimit
	return testChain, err
}

// initializeCoverageMaps initializes our coverage maps by tracing initial deployments in the test chain and replaying
// all call sequences stored in the corpus.
func (f *Fuzzer) initializeCoverageMaps(baseTestChain *chain.TestChain) error {
	// Create our coverage maps and a coverage tracer
	f.coverageMaps = coverage.NewCoverageMaps()
	coverageTracer := coverage.NewCoverageTracer()

	// Clone our test chain with our coverage tracer.
	testChain, err := baseTestChain.Clone([]vm.EVMLogger{coverageTracer}, nil)
	if err != nil {
		return fmt.Errorf("failed to initialize coverage maps, base test chain cloning encountered error: %v", err)
	}

	// Next we measure coverage for every corpus call sequence.
	corpusCallSequences := f.corpus.CallSequences()

	// Cache current HeadBlockNumber so that you can reset back to it after every sequence
	baseBlockNumber := testChain.HeadBlockNumber()

	for _, sequence := range corpusCallSequences {
		// Execute each call sequence, collecting coverage and updating it along the way
		_, err = sequence.ExecuteOnChain(testChain, true, nil, func(index int) (bool, error) {
			// Update our coverage maps for each call executed in our sequence.
			covMaps := coverage.GetCoverageTracerResults(sequence[index].ChainReference.MessageResults())
			_, covErr := f.coverageMaps.Update(covMaps)
			if covErr != nil {
				return true, fmt.Errorf("coverage map update encountered error: %v", err)
			}
			return false, nil
		})
		if err != nil {
			return fmt.Errorf("failed to initialize coverage maps, encountered an error while executing call sequence: %v\n", err)
		}

		// Revert chain state to our starting point to test the next sequence.
		err = testChain.RevertToBlockNumber(baseBlockNumber)
		if err != nil {
			return fmt.Errorf("failed to reset the chain while seeding coverage: %v\n", err)
		}
	}
	return nil
}

// chainSetupFromCompilations is a TestChainSetupFunc which sets up the base test chain state by deploying
// all compiled contract definitions. This includes any successful compilations as a result of the Fuzzer.config
// definitions, as well as those added by Fuzzer.AddCompilationTargets. The contract deployment order is defined by
// the Fuzzer.config.
func chainSetupFromCompilations(fuzzer *Fuzzer, testChain *chain.TestChain) error {
	// Verify contract deployment order is not empty.
	if len(fuzzer.config.Fuzzing.DeploymentOrder) == 0 {
		return fmt.Errorf("you must specify a contract deployment order within your project configuration")
	}

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

// defaultNewValueGeneratorFunc is a NewValueGeneratorFunc which creates a valuegeneration.MutatingValueGenerator with a
// default configuration. Returns the generator or an error, if one occurs.
func defaultNewValueGeneratorFunc(fuzzer *Fuzzer, valueSet *valuegeneration.ValueSet) (valuegeneration.ValueGenerator, error) {
	valueGenConfig := &valuegeneration.MutatingValueGeneratorConfig{
		MinMutationRounds: 0,
		MaxMutationRounds: 1,
		RandomAddressBias: 0.5,
		RandomIntegerBias: 0.5,
		RandomStringBias:  0.5,
		RandomBytesBias:   0.5,
		RandomValueGeneratorConfig: &valuegeneration.RandomValueGeneratorConfig{
			RandomArrayMinSize:  0,
			RandomArrayMaxSize:  100,
			RandomBytesMinSize:  0,
			RandomBytesMaxSize:  100,
			RandomStringMinSize: 0,
			RandomStringMaxSize: 100,
		},
	}
	valueGenerator := valuegeneration.NewMutatingValueGenerator(valueGenConfig, valueSet)
	return valueGenerator, nil
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
	availableWorkerIndexes := make([]int, f.config.Fuzzing.Workers)
	availableWorkerIndexedLock := sync.Mutex{}
	for i := 0; i < len(availableWorkerIndexes); i++ {
		availableWorkerIndexes[i] = i
	}

	// Define a flag that indicates whether we have not cancelled o
	working := !utils.CheckContextDone(f.ctx)

	// Log that we are about to create the workers and start fuzzing
	fmt.Printf("Creating %d workers ...\n", f.config.Fuzzing.Workers)
	var err error
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
			availableWorkerIndexes = append(availableWorkerIndexes, workerIndex)
			availableWorkerIndexedLock.Unlock()

			// Publish an event indicating we destroyed a worker.
			workerDestroyedErr := f.Events.WorkerDestroyed.Publish(FuzzerWorkerDestroyedEvent{Worker: worker})
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
	return err
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
	err = f.Hooks.ChainSetupFunc(f, baseTestChain)
	if err != nil {
		return err
	}

	// Initialize our coverage maps
	err = f.initializeCoverageMaps(baseTestChain)
	if err != nil {
		return err
	}

	// Start our printing loop now that we're about to begin fuzzing.
	go f.runMetricsPrintLoop()

	// Publish a fuzzer starting event.
	err = f.Events.FuzzerStarting.Publish(FuzzerStartingEvent{Fuzzer: f})
	if err != nil {
		return err
	}

	// Run the main worker loop
	err = f.spawnWorkersLoop(baseTestChain)

	// NOTE: After this point, we capture errors but do not return immediately, as we want to exit gracefully.

	// If we have coverage enabled and a corpus directory set, write the corpus. We do this even if we had a
	// previous error, as we don't want to lose corpus entries.
	if f.config.Fuzzing.CoverageEnabled {
		corpusFlushErr := f.corpus.Flush()
		if err == nil {
			err = corpusFlushErr
		}
	}

	// Publish a fuzzer stopping event.
	fuzzerStoppingErr := f.Events.FuzzerStopping.Publish(FuzzerStoppingEvent{Fuzzer: f, err: err})
	if err == nil && fuzzerStoppingErr != nil {
		err = fuzzerStoppingErr
	}

	// Print our test case results
	fmt.Printf("\n")
	fmt.Printf("Fuzzer stopped, test results follow below ...\n")
	for _, testCase := range f.testCases {
		// Obtain the test case message. If it is a non-empty string, we format our output for it specially.
		// Otherwise, we exclude it.
		msg := strings.TrimSpace(testCase.Message())
		if msg != "" {
			fmt.Printf("[%s] %s\n%s\n\n", testCase.Status(), strings.TrimSpace(testCase.Name()), msg)
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
	var lastCallsTested, lastSequencesTested, lastWorkerStartupCount uint64
	lastPrintedTime := time.Time{}
	for !utils.CheckContextDone(f.ctx) {
		// Obtain our metrics
		callsTested := f.metrics.CallsTested()
		sequencesTested := f.metrics.SequencesTested()
		workerStartupCount := f.metrics.WorkerStartupCount()

		// Calculate time elapsed since the last update
		secondsSinceLastUpdate := time.Now().Sub(lastPrintedTime).Seconds()

		// Print a metrics update
		fmt.Printf(
			"fuzz: elapsed: %s, call: %d (%d/sec), seq/s: %d, worker resets: %d/s\n",
			time.Now().Sub(startTime).Round(time.Second),
			callsTested,
			uint64(float64(callsTested-lastCallsTested)/secondsSinceLastUpdate),
			uint64(float64(sequencesTested-lastSequencesTested)/secondsSinceLastUpdate),
			uint64(float64(workerStartupCount-lastWorkerStartupCount)/secondsSinceLastUpdate),
		)

		// Update our delta tracking metrics
		lastPrintedTime = time.Now()
		lastCallsTested = callsTested
		lastSequencesTested = sequencesTested
		lastWorkerStartupCount = workerStartupCount

		// If we reached our transaction threshold, halt
		testLimit := f.config.Fuzzing.TestLimit
		if testLimit > 0 && callsTested >= testLimit {
			fmt.Printf("transaction test limit reached, halting now ...\n")
			f.Stop()
			break
		}

		// Sleep for a second
		time.Sleep(time.Second * 3)
	}
}
