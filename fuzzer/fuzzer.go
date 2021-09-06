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
)

type Fuzzer struct {
	// config describes the project configuration which the fuzzer is targeting.
	config   configs.ProjectConfig
	// accounts describes a set of account keys derived from config, for use in fuzzing campaigns.
	accounts []fuzzerAccount


	// ctx describes the context for the fuzzer run, used to cancel running operations.
	ctx context.Context
	// ctxCancelFunc describes a function which can be used to cancel the fuzzing operations ctx tracks.
	ctxCancelFunc context.CancelFunc
	// compilations describes the compiled targets produced by the last Start call for the Fuzzer to target.
	compilations []types.Compilation

}

type fuzzerAccount struct {
	// key describes the ecdsa private key of an account used a Fuzzer instance.
	key *ecdsa.PrivateKey
	// address represents the ethereum address which corresponds to key.
	address common.Address
}

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

	// Create a test node for each thread we intend to create.
	fmt.Printf("Creating %d test node threads ...\n", f.config.ThreadCount)
	threadReserveChannel := make(chan struct{}, f.config.ThreadCount)
	for err == nil {
		// Send an item into our channel to queue up a spot
		threadReserveChannel <- struct{}{}

		// Run our goroutine. This should take our queued struct out of the channel once it's done,
		// keeping us at our desired thread capacity.
		go func() {
			// Create a new worker for this fuzzer and run it.
			worker := newFuzzWorker(f)
			err = worker.run()

			// Free up a thread in our reserved thread channel.
			<- threadReserveChannel
		}()
	}
	return err
}

// Stop stops all working goroutines in the Fuzzer
func (f *Fuzzer) Stop() {
	// Call the cancel function on our running context to stop all working goroutines
	if f.ctxCancelFunc != nil {
		f.ctxCancelFunc()
	}
}