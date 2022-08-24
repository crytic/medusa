package fuzzing

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/trailofbits/medusa/compilation"
	"github.com/trailofbits/medusa/compilation/types"
	"github.com/trailofbits/medusa/configs"
	fuzzerTypes "github.com/trailofbits/medusa/fuzzing/types"
	"github.com/trailofbits/medusa/fuzzing/value_generation"
	"github.com/trailofbits/medusa/utils"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Fuzzer represents an Ethereum smart contract fuzzing provider.
type Fuzzer struct {
	// config describes the project configuration which the fuzzing is targeting.
	config configs.ProjectConfig
	// accounts describes a set of account keys derived from config, for use in fuzzing campaigns.
	accounts []common.Address

	// ctx describes the context for the fuzzing run, used to cancel running operations.
	ctx context.Context
	// ctxCancelFunc describes a function which can be used to cancel the fuzzing operations ctx tracks.
	ctxCancelFunc context.CancelFunc
	// compilations describes the compiled targets produced by the last Start call for the Fuzzer to target.
	compilations []types.Compilation

	// baseValueSet represents a base_value_set.BaseValueSet containing input values for our fuzz tests.
	baseValueSet *value_generation.BaseValueSet
	// generator defines our fuzzing approach to generate function inputs.
	generator value_generation.ValueGenerator
	// workers represents the work threads created by this Fuzzer when Start invokes a fuzz operation.
	workers []*fuzzerWorker
	// metrics represents the metrics for the fuzzing campaign.
	metrics *FuzzerMetrics
	// results describes the results we track during our fuzzing campaign, such as failed property tests.
	results *FuzzerResults
	// corpus stores a list of transaction sequences that can be used for coverage-guided fuzzing
	corpus *fuzzerTypes.Corpus
}

// NewFuzzer returns an instance of a new Fuzzer provided a project configuration, or an error if one is encountered
// while initializing the code.
func NewFuzzer(config configs.ProjectConfig) (*Fuzzer, error) {
	// Create our accounts based on our configs
	accounts := make([]common.Address, 0)

	// Set up accounts for provided keys
	for i := 0; i < len(config.Accounts.Predefined); i++ {
		// Parse our provided account string
		address, err := utils.HexStringToAddress(config.Accounts.Predefined[i])
		if err != nil {
			return nil, err
		}

		// Add it to our account list
		accounts = append(accounts, *address)
	}

	// Generate new accounts as requested.
	for i := 0; i < config.Accounts.Generate; i++ {
		// Generate a new key
		key, err := crypto.GenerateKey()
		if err != nil {
			return nil, err
		}

		// Add it to our account list
		accounts = append(accounts, crypto.PubkeyToAddress(key.PublicKey))
	}

	// Print some updates regarding account keys loaded
	fmt.Printf("Account keys loaded (%d generated, %d pre-defined) ...\n", config.Accounts.Generate, len(config.Accounts.Predefined))
	for i := 0; i < len(accounts); i++ {
		accountAddr := accounts[i].String()
		fmt.Printf("-[account #%d] address=%s\n", i+1, accountAddr)
	}

	// Create and return our fuzzing instance.
	fuzzer := &Fuzzer{
		config:   config,
		accounts: accounts,
	}
	return fuzzer, nil
}

// Results exposes the underlying results of the fuzzer, including any violated property tests and transaction sequences
// used to do so.
func (f *Fuzzer) Results() *FuzzerResults {
	return f.results
}

// Start begins a fuzzing operation on the provided project configuration. This operation will not return until an error
// is encountered or the fuzzing operation has completed. Its execution can be cancelled using the Stop method.
// Returns an error if one is encountered.
func (f *Fuzzer) Start() error {
	// Create our running context (allows us to cancel across threads)
	f.ctx, f.ctxCancelFunc = context.WithCancel(context.Background())

	// Compile our targets
	fmt.Printf("Compiling targets (platform '%s') ...\n", f.config.Compilation.Platform)
	compilations, compilationOutput, err := compilation.Compile(f.config.Compilation)
	f.compilations = compilations
	if err != nil {
		return err
	}
	fmt.Printf(compilationOutput)

	// We create a test node for each thread we intend to create. Fuzzer workers can stop if they hit some resource
	// limit such as a memory limit, at which point we'll recreate them in our loop, putting them into the same index.
	// For now, we create our available index queue before initializing some providers and entering our main loop.
	fmt.Printf("Creating %d workers ...\n", f.config.Fuzzing.Workers)
	availableWorkerIndexes := make([]int, f.config.Fuzzing.Workers)
	availableWorkerIndexedLock := sync.Mutex{}
	for i := 0; i < len(availableWorkerIndexes); i++ {
		availableWorkerIndexes[i] = i
	}

	// Initialize our BaseValueSet and seed it from values derived from ASTs
	f.baseValueSet = value_generation.NewBaseValueSet()
	for _, c := range f.compilations {
		for _, source := range c.Sources {
			f.baseValueSet.SeedFromAst(source.Ast)
		}
	}

	// Initialize generator, results, and metrics providers.
	f.results = NewFuzzerResults()
	f.metrics = newFuzzerMetrics(f.config.Fuzzing.Workers)
	go f.runMetricsPrintLoop()
	f.generator = value_generation.NewValueGeneratorMutation(f.baseValueSet) // TODO: make this configurable after adding more options

	// Setup corpus
	if f.config.Fuzzing.Coverage {
		f.corpus = fuzzerTypes.NewCorpus()
	}
	if f.config.Fuzzing.Coverage && f.config.Fuzzing.CorpusDirectory != "" {
		_, err = f.checkAndSetupCorpusDirectory()
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}

	// If we set a timeout, create the timeout context now, as we're about to begin fuzzing.
	if f.config.Fuzzing.Timeout > 0 {
		fmt.Printf("Running with timeout of %d seconds\n", f.config.Fuzzing.Timeout)
		f.ctx, f.ctxCancelFunc = context.WithTimeout(f.ctx, time.Duration(f.config.Fuzzing.Timeout)*time.Second)
	}

	// Finally, we create our fuzz workers in a loop, using a channel to block when we reach capacity.
	// If we encounter any errors, we stop.
	f.workers = make([]*fuzzerWorker, f.config.Fuzzing.Workers)
	threadReserveChannel := make(chan struct{}, f.config.Fuzzing.Workers)
	ctxAlive := true
	for err == nil && ctxAlive {
		// Send an item into our channel to queue up a spot. This will block us if we hit capacity until a worker
		// slot is freed up.
		threadReserveChannel <- struct{}{}

		// Pop a worker index off of our queue
		availableWorkerIndexedLock.Lock()
		workerIndex := availableWorkerIndexes[0]
		availableWorkerIndexes = availableWorkerIndexes[1:]
		availableWorkerIndexedLock.Unlock()

		// Run our goroutine. This should take our queued struct out of the channel once it's done,
		// keeping us at our desired thread capacity.
		go func(workerIndex int) {
			// Create a new worker for this fuzzing.
			worker := newFuzzerWorker(f, workerIndex)
			f.workers[workerIndex] = worker

			// Run the fuzz worker and set our result such that errors or a ctx cancellation will exit the loop.
			ctxCancelled, workerErr := worker.run()
			if workerErr != nil {
				err = workerErr
			}
			ctxAlive = ctxAlive && !ctxCancelled

			// Free our worker id before unblocking our channel, as a free one will be expected.
			availableWorkerIndexedLock.Lock()
			availableWorkerIndexes = append(availableWorkerIndexes, workerIndex)
			availableWorkerIndexedLock.Unlock()

			// Unblock our channel by freeing our capacity of another item, making way for another worker.
			<-threadReserveChannel
		}(workerIndex)
	}

	// Return any encountered error.
	return err
}

// Stop stops a running operation invoked by the Start method. This method may return before complete operation teardown
// occurs.
func (f *Fuzzer) Stop() {
	// Write corpus to disk if corpusDirectory is set and coverage is enabled
	if f.config.Fuzzing.CorpusDirectory != "" && f.config.Fuzzing.Coverage {
		err := f.writeCorpusToDisk()
		// TODO: Should I throw a panic?
		if err != nil {
			panic(err)
		}
	}

	// Call the cancel function on our running context to stop all working goroutines
	if f.ctxCancelFunc != nil {
		f.ctxCancelFunc()
	}
}

// checkAndSetupCorpusDirectory sets up the my_corpus/ directory and subdirectories. If the (sub)directories already exist
// then return true so that corpus can be read into memory. Note that it will return true even if my_corpus/corpus is empty.
// The function checks for directory existence and nothing more.
// TODO: Are the directory permissions too lose? 0644 and 0666 did would lead to permission errors
// TODO: Should there be a regex check to make sure it is a valid directory name?
func (f *Fuzzer) checkAndSetupCorpusDirectory() (bool, error) {
	// Get info on corpus_dir directory existence
	dirInfo, err := os.Stat(f.config.Fuzzing.CorpusDirectory)
	if err != nil {
		// If directory does not exist, make 'corpus', 'corpus/corpus', and 'corpus/coverage'
		if os.IsNotExist(err) {
			if err = os.Mkdir(f.config.Fuzzing.CorpusDirectory, 0777); err != nil {
				return false, fmt.Errorf("error while creating corpus directory. Make sure the config_dir config option is a valid directory name: %v\n", err)
			}
			if err = os.Mkdir(filepath.Join(f.config.Fuzzing.CorpusDirectory, "/corpus"), 0777); err != nil {
				return false, fmt.Errorf("error while creating corpus sub-directory: %v\n", err)
			}
			if err = os.Mkdir(filepath.Join(f.config.Fuzzing.CorpusDirectory, "/coverage"), 0777); err != nil {
				return false, fmt.Errorf("error while creating coverage sub-directory: %v\n", err)
			}
			return false, nil
		}

		// some other sort of error, throw it
		return false, err
	}

	// if corpus is a file and not a directory, throw an error
	if !dirInfo.IsDir() {
		return false, fmt.Errorf("there exists a conflicting file named %s in this directory.\n", f.config.Fuzzing.CorpusDirectory)
	}

	// If corpus/corpus is not there, make it
	if _, err = os.Stat(f.config.Fuzzing.CorpusDirectory + "/corpus"); os.IsNotExist(err) {
		if err = os.Mkdir(filepath.Join(f.config.Fuzzing.CorpusDirectory, "/corpus"), 0777); err != nil {
			return false, fmt.Errorf("error while creating corpus sub-directory: %v\n", err)
		}
	}

	// if corpus/coverage is not there, make it
	if _, err = os.Stat(f.config.Fuzzing.CorpusDirectory + "/coverage"); os.IsNotExist(err) {
		if err = os.Mkdir(filepath.Join(f.config.Fuzzing.CorpusDirectory, "/coverage"), 0777); err != nil {
			return false, fmt.Errorf("error while creating coverage sub-directory: %v\n", err)
		}
	}

	// we will return true even if corpus/corpus is empty.
	// There is no reason to check here whether there are files in there right now.
	return true, nil
}

// runMetricsPrintLoop prints metrics to the console in a loop until ctx signals a stopped operation.
func (f *Fuzzer) runMetricsPrintLoop() {
	// Define cached variables for our metrics to calculate deltas.
	var lastTransactionsTested, lastSequencesTested, lastWorkerStartupCount uint64
	lastPrintedTime := time.Time{}
	for {
		// Obtain our metrics
		transactionsTested := f.metrics.TransactionsTested()
		sequencesTested := f.metrics.SequencesTested()
		workerStartupCount := f.metrics.WorkerStartupCount()

		// Calculate time elapsed since the last update
		secondsSinceLastUpdate := time.Now().Sub(lastPrintedTime).Seconds()

		// Print a metrics update
		fmt.Printf(
			"tx num: %d, workers: %d, hitmemlimit: %d/s, tx/s: %d, seq/s: %d\n",
			transactionsTested,
			len(f.metrics.workerMetrics),
			uint64(float64(workerStartupCount-lastWorkerStartupCount)/secondsSinceLastUpdate),
			uint64(float64(transactionsTested-lastTransactionsTested)/secondsSinceLastUpdate),
			uint64(float64(sequencesTested-lastSequencesTested)/secondsSinceLastUpdate),
		)

		// Update our delta tracking metrics
		lastPrintedTime = time.Now()
		lastTransactionsTested = transactionsTested
		lastSequencesTested = sequencesTested
		lastWorkerStartupCount = workerStartupCount

		// If we reached our transaction threshold, halt
		testLimit := uint64(f.config.Fuzzing.TestLimit)
		if testLimit > 0 && transactionsTested >= testLimit {
			fmt.Printf("transaction test limit reached, halting now ...\n")
			f.Stop()
		}

		// Sleep for a second
		time.Sleep(time.Second)

		// If ctx signalled to stop the operation, return immediately.
		select {
		case <-f.ctx.Done():
			return
		default:
			break
		}
	}
}

// writeCorpusToDisk will write the corpus to disk after the fuzzer finishes its work
func (f *Fuzzer) writeCorpusToDisk() error {
	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}
	// Move to corpus/corpus subdirectory
	err = os.Chdir(filepath.Join(currentDir, f.config.Fuzzing.CorpusDirectory, "/corpus"))
	if err != nil {
		return err
	}
	// Write all sequences to corpus
	for hash, corpusBlockSequence := range f.corpus.CorpusBlockSequences {
		fileName := hash + ".json"
		// If corpus file already exists, no need to write it again
		if _, err := os.Stat(fileName); err == nil {
			continue
		}
		// Marshal the sequence
		jsonString, err := json.MarshalIndent(corpusBlockSequence, "", " ")
		if err != nil {
			return err
		}
		// Write the byte string
		err = ioutil.WriteFile(fileName, jsonString, os.ModePerm)
		if err != nil {
			return err
		}
	}
	// Change back to original directory
	err = os.Chdir(currentDir)
	if err != nil {
		return err
	}

	return nil
}
