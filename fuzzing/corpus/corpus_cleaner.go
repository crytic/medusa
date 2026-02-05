package corpus

import (
	"context"
	"fmt"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/logging"
)

// CorpusCleaner provides functionality to clean invalid sequences from a corpus.
// It follows the same pattern as CorpusPruner by not depending on the Fuzzer type.
type CorpusCleaner struct {
	// corpus is the corpus to be cleaned
	corpus *Corpus
	// logger is used to log when cleaning and on error
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

// NewCorpusCleaner creates a new CorpusCleaner.
func NewCorpusCleaner(corpus *Corpus, logger *logging.Logger) *CorpusCleaner {
	return &CorpusCleaner{
		corpus: corpus,
		logger: logger,
	}
}

// Clean validates call sequences using the provided test chain and deployed contracts.
// Sequences that fail to execute are removed from disk (unless dryRun is true).
// Returns the cleaning results and any error encountered.
func (cc *CorpusCleaner) Clean(
	ctx context.Context,
	testChain *chain.TestChain,
	deployedContracts map[common.Address]*contracts.Contract,
	dryRun bool,
) (*CleanResult, error) {
	// Get base block index for reverting
	chainBaseIndex := uint64(len(testChain.CommittedBlocks()))

	// Use the corpus's cleaning method
	cleanResult, err := cc.corpus.CleanInvalidSequences(ctx, testChain, deployedContracts, dryRun)
	if err != nil {
		return nil, fmt.Errorf("error during corpus cleaning: %w", err)
	}

	// Revert to base state
	_ = testChain.RevertToBlockIndex(chainBaseIndex)

	return &CleanResult{
		TotalSequences:   cleanResult.TotalSequences,
		ValidSequences:   cleanResult.ValidSequences,
		InvalidSequences: cleanResult.InvalidSequences,
	}, nil
}
