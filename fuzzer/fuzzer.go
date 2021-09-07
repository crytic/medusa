package fuzzer

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"medusa/compilation"
	"medusa/compilation/types"
	"medusa/configs"
	"sync"
	"time"
)

// Fuzzer represents an Ethereum smart contract fuzzing provider.
type Fuzzer struct {
	// config describes the project configuration which the fuzzer is targeting.
	config configs.ProjectConfig
	// accounts describes a set of account keys derived from config, for use in fuzzing campaigns.
	accounts []fuzzerAccount


	// ctx describes the context for the fuzzer run, used to cancel running operations.
	ctx context.Context
	// ctxCancelFunc describes a function which can be used to cancel the fuzzing operations ctx tracks.
	ctxCancelFunc context.CancelFunc
	// compilations describes the compiled targets produced by the last Start call for the Fuzzer to target.
	compilations []types.Compilation

	// generator defines our fuzzing approach to generate transactions.
	generator txGenerator
	// workers represents the work threads created by this Fuzzer when Start invokes a fuzz operation.
	workers []*fuzzerWorker
	// metrics represents the metrics for the fuzzing campaign.
	metrics *FuzzerMetrics
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

	// Generate new accounts as requested.
	for i := 0; i < config.Accounts.Generate; i++ {
		// Generate a new key
		key, err := crypto.GenerateKey()
		if err != nil {
			return nil, err
		}

		// Add it to our account list
		acc := fuzzerAccount{
			key: key,
			address: crypto.PubkeyToAddress(key.PublicKey),
		}
		accounts = append(accounts, acc)
	}

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
			key: key,
			address: crypto.PubkeyToAddress(key.PublicKey),
		}
		accounts = append(accounts, acc)
	}

	// Print some output
	fmt.Printf("Account keys loaded (%d generated, %d pre-defined) ...\n", config.Accounts.Generate, len(config.Accounts.Keys))

	// Create and return our fuzzer instance.
	fuzzer := &Fuzzer{
		config: config,
		accounts: accounts,
	}
	return fuzzer, nil
}

// Start begins a fuzzing operation on the provided project configuration. This operation will not return until an error
// is encountered or the fuzzing operation has completed. Its execution can be cancelled using the Stop method.
// Returns an error if one is encountered.
func (f *Fuzzer) Start() error {
	// Create our running context
	f.ctx, f.ctxCancelFunc = context.WithCancel(context.Background())

	// Compile our targets
	var err error
	fmt.Printf("Compiling targets (platform '%s') ...\n", f.config.Compilation.Platform)
	f.compilations, err = compilation.Compile(f.config.Compilation)
	if err != nil {
		return err
	}

	// Create a test node for each thread we intend to create. Fuzzer workers can stop if they hit some resource
	// limit such as a memory limit, at which point we'll recreate them here, putting them into the same index.
	// First, create the available index queue.
	fmt.Printf("Creating %d workers ...\n", f.config.Fuzzing.Workers)
	availableWorkerIndexes := make([]int, f.config.Fuzzing.Workers)
	availableWorkerIndexedLock := sync.Mutex{}
	for i := 0; i < len(availableWorkerIndexes); i++ { availableWorkerIndexes[i] = i }

	// Next, initialize our metrics and generator
	f.metrics = NewFuzzerMetrics(f.config.Fuzzing.Workers)
	go f.runMetricsPrintLoop()
	f.generator = newTxGeneratorRandom() // TODO: make this configurable after adding more options

	// Finally, we create our fuzz workers in a loop, using a channel to block when we reach capacity.
	// If we encounter any errors, we stop.
	f.workers = make([]*fuzzerWorker, f.config.Fuzzing.Workers)
	threadReserveChannel := make(chan struct{}, f.config.Fuzzing.Workers)
	for err == nil {
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
			// Create a new worker for this fuzzer and run it.
			worker := newFuzzerWorker(workerIndex, f)
			f.workers[workerIndex] = worker
			err = worker.run()

			// Free our worker id before unblocking our channel, as a free one will be expected.
			availableWorkerIndexedLock.Lock()
			availableWorkerIndexes = append(availableWorkerIndexes, workerIndex)
			availableWorkerIndexedLock.Unlock()

			// Unblock our channel by freeing our capacity of another item, making way for another worker.
			<- threadReserveChannel
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
	var lastTransactionsTested, lastSequencesTested uint64
	for {
		// Obtain our metrics
		transactionsTested := f.metrics.TransactionsTested()
		sequencesTested := f.metrics.SequencesTested()

		// Print a metrics update
		fmt.Printf(
			"workers: %d, tx/s: %d, seq/s: %d\n",
			len(f.metrics.workerMetrics),
			transactionsTested - lastTransactionsTested,
			sequencesTested - lastSequencesTested,
			)

		// Update our delta tracking metrics
		lastTransactionsTested = transactionsTested
		lastSequencesTested = sequencesTested

		// Sleep for two seconds
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