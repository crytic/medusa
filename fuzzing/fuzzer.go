package fuzzing

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/keycard-go/hexutils"
	"github.com/trailofbits/medusa/compilation"
	"github.com/trailofbits/medusa/compilation/types"
	"github.com/trailofbits/medusa/configs"
	"github.com/trailofbits/medusa/fuzzing/value_generation"
	"sync"
	"time"
)

// Fuzzer represents an Ethereum smart contract fuzzing provider.
type Fuzzer struct {
	// config describes the project configuration which the fuzzing is targeting.
	config configs.ProjectConfig
	// accounts describes a set of account keys derived from config, for use in fuzzing campaigns.
	accounts []fuzzerAccount

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
}

// fuzzerAccount represents a single keypair generated or derived from settings provided in the Fuzzer.config.
type fuzzerAccount struct {
	// key describes the ecdsa private key of an account used a Fuzzer instance.
	key *ecdsa.PrivateKey
	// address represents the ethereum address which corresponds to key.
	address common.Address
}

// NewFuzzer returns an instance of a new Fuzzer provided a project configuration, or an error if one is encountered
// while initializing the code.
func NewFuzzer(config configs.ProjectConfig) (*Fuzzer, error) {
	// Create our accounts based on our configs
	accounts := make([]fuzzerAccount, 0)

	// Set up accounts for provided keys
	for i := 0; i < len(config.Accounts.Keys); i++ {
		// Parse our provided key string
		keyStr := config.Accounts.Keys[i]
		key, err := crypto.HexToECDSA(keyStr)
		if err != nil {
			return nil, err
		}

		// Add it to our account list
		acc := fuzzerAccount{
			key:     key,
			address: crypto.PubkeyToAddress(key.PublicKey),
		}
		accounts = append(accounts, acc)
	}

	// Generate new accounts as requested.
	for i := 0; i < config.Accounts.Generate; i++ {
		// Generate a new key
		key, err := crypto.GenerateKey()
		if err != nil {
			return nil, err
		}

		// Add it to our account list
		acc := fuzzerAccount{
			key:     key,
			address: crypto.PubkeyToAddress(key.PublicKey),
		}
		accounts = append(accounts, acc)
	}

	// Print some updates regarding account keys loaded
	fmt.Printf("Account keys loaded (%d generated, %d pre-defined) ...\n", config.Accounts.Generate, len(config.Accounts.Keys))

	for i := 0; i < len(accounts); i++ {
		accountAddr := crypto.PubkeyToAddress(accounts[i].key.PublicKey).String()
		accountKey := hexutils.BytesToHex(crypto.FromECDSA(accounts[i].key))
		fmt.Printf("-[account #%d] address=%s, key=%s\n", i+1, accountAddr, accountKey)
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
	// Create our running context
	if f.config.Fuzzing.Timeout > 0 {
		fmt.Printf("Running with timeout of %d seconds\n", f.config.Fuzzing.Timeout)
		f.ctx, f.ctxCancelFunc = context.WithTimeout(context.Background(), time.Duration(f.config.Fuzzing.Timeout)*time.Second)
	} else {
		f.ctx, f.ctxCancelFunc = context.WithCancel(context.Background())
	}

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
	// Call the cancel function on our running context to stop all working goroutines
	if f.ctxCancelFunc != nil {
		f.ctxCancelFunc()
	}
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
