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
	"github.com/rs/zerolog"
)

// CorpusCleaner provides functionality to clean invalid sequences from a corpus.
type CorpusCleaner struct {
	fuzzer *Fuzzer
	logger *logging.Logger
}

// CleanResult contains the results of a corpus cleaning operation.
type CleanResult struct {
	// TotalSequences is the total number of sequences in the corpus before cleaning.
	TotalSequences int
	// ValidSequences is the number of sequences that executed successfully.
	ValidSequences int
	// InvalidSequences is the list of filenames that were invalid.
	InvalidSequences []string
}

// NewCorpusCleaner creates a new CorpusCleaner from the project config.
// It initializes the underlying fuzzer to set up compilation and contract definitions.
func NewCorpusCleaner(fuzzer *Fuzzer) *CorpusCleaner {
	return &CorpusCleaner{
		fuzzer: fuzzer,
		logger: logging.NewLogger(zerolog.InfoLevel),
	}
}

// Clean validates each call sequence in the corpus by attempting to execute it.
// Sequences that fail are considered invalid and removed from disk (unless dryRun is true).
// Returns the cleaning results and any error encountered.
func (cc *CorpusCleaner) Clean(ctx context.Context, dryRun bool) (*CleanResult, error) {
	result := &CleanResult{
		InvalidSequences: make([]string, 0),
	}

	// Create a test chain for validation
	baseTestChain, err := cc.fuzzer.createTestChain()
	if err != nil {
		return nil, fmt.Errorf("failed to create test chain: %w", err)
	}
	defer baseTestChain.Close()

	// Set up the chain with contract deployments
	_, err = cc.fuzzer.Hooks.ChainSetupFunc(cc.fuzzer, baseTestChain)
	if err != nil {
		return nil, fmt.Errorf("failed to setup test chain: %w", err)
	}

	// Build the deployed contracts map by cloning the chain and subscribing to events
	// This is the same pattern used by corpus.Initialize
	deployedContracts := make(map[common.Address]*fuzzerTypes.Contract)
	testChain, err := baseTestChain.Clone(func(newChain *chain.TestChain) error {
		// Subscribe to contract deployment events to track deployed contracts
		newChain.Events.ContractDeploymentAddedEventEmitter.Subscribe(
			func(event chain.ContractDeploymentsAddedEvent) error {
				matchedContract := cc.fuzzer.contractDefinitions.MatchBytecode(
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

	// Create and load the corpus
	corpusDir := cc.fuzzer.config.Fuzzing.CorpusDirectory
	c, err := corpus.NewCorpus(corpusDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load corpus: %w", err)
	}

	// Get the base block index for reverting
	chainBaseIndex := uint64(len(testChain.CommittedBlocks()))

	// Process each call sequence file
	result.TotalSequences, _ = c.CallSequenceEntryCount()

	// Use the corpus's internal cleaning method
	cleanResult, err := c.CleanInvalidSequences(ctx, testChain, deployedContracts, dryRun)
	if err != nil {
		return nil, fmt.Errorf("error during corpus cleaning: %w", err)
	}

	result.TotalSequences = cleanResult.TotalSequences
	result.ValidSequences = cleanResult.ValidSequences
	result.InvalidSequences = cleanResult.InvalidSequences

	// Revert to base state (in case it wasn't done)
	_ = testChain.RevertToBlockIndex(chainBaseIndex)

	return result, nil
}

// CleanCorpus is a convenience function that creates a fuzzer from the config,
// then cleans the corpus. This is the main entry point for the CLI command.
func CleanCorpus(
	ctx context.Context,
	fuzzer *Fuzzer,
	dryRun bool,
	logger *logging.Logger,
) (*CleanResult, error) {
	// Check if corpus directory exists
	if fuzzer.config.Fuzzing.CorpusDirectory == "" {
		return nil, fmt.Errorf("no corpus directory configured")
	}

	cleaner := NewCorpusCleaner(fuzzer)

	start := time.Now()
	result, err := cleaner.Clean(ctx, dryRun)
	if err != nil {
		return nil, err
	}

	logger.Info("Corpus cleaning completed in ", time.Since(start).Round(time.Second))

	return result, nil
}
