package fuzzing

import (
	"context"
	"fmt"
	"time"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa/chain"
	fuzzerTypes "github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/fuzzing/corpus"
	"github.com/crytic/medusa/logging"
)

// CleanResult contains the results of a corpus cleaning operation.
type CleanResult = corpus.CleanResult

// CleanCorpus is a convenience function for the CLI that sets up a test chain and cleans the corpus.
// It validates each call sequence in the corpus by attempting to execute it on a test chain.
// Sequences that fail to execute are removed from disk (unless dryRun is true).
// Returns the cleaning results and any error encountered.
func CleanCorpus(
	ctx context.Context,
	fuzzer *Fuzzer,
	dryRun bool,
	logger *logging.Logger,
) (*CleanResult, error) {
	// Check if corpus directory is configured
	if fuzzer.config.Fuzzing.CorpusDirectory == "" {
		return nil, fmt.Errorf("no corpus directory configured")
	}

	// Create test chain
	baseTestChain, err := fuzzer.createTestChain()
	if err != nil {
		return nil, fmt.Errorf("failed to create test chain: %w", err)
	}
	defer baseTestChain.Close()

	// Setup chain
	_, err = fuzzer.Hooks.ChainSetupFunc(fuzzer, baseTestChain)
	if err != nil {
		return nil, fmt.Errorf("failed to setup test chain: %w", err)
	}

	// Build deployed contracts map by cloning the chain and subscribing to events
	deployedContracts := make(map[common.Address]*fuzzerTypes.Contract)
	testChain, err := baseTestChain.Clone(func(newChain *chain.TestChain) error {
		// Subscribe to contract deployment events to track deployed contracts
		newChain.Events.ContractDeploymentAddedEventEmitter.Subscribe(
			func(event chain.ContractDeploymentsAddedEvent) error {
				matchedContract := fuzzer.contractDefinitions.MatchBytecode(
					event.Contract.InitBytecode,
					event.Contract.RuntimeBytecode,
				)
				if matchedContract != nil {
					deployedContracts[event.Contract.Address] = matchedContract
				}
				return nil
			},
		)
		newChain.Events.ContractDeploymentRemovedEventEmitter.Subscribe(
			func(event chain.ContractDeploymentsRemovedEvent) error {
				delete(deployedContracts, event.Contract.Address)
				return nil
			},
		)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to clone test chain: %w", err)
	}
	defer testChain.Close()

	// Create cleaner and run
	cleaner := corpus.NewCorpusCleaner(fuzzer.corpus, logger)
	start := time.Now()
	result, err := cleaner.Clean(ctx, testChain, deployedContracts, dryRun)
	if err != nil {
		return nil, err
	}

	logger.Info("Corpus cleaning completed in ", time.Since(start).Round(time.Second))
	return result, nil
}
