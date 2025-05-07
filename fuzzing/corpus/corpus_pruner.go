package corpus

import (
	"context"
	"fmt"
	"time"

	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/fuzzing/coverage"
	"github.com/crytic/medusa/logging"
	"github.com/crytic/medusa/utils"
)

// CorpusPruner is a job that runs every `PruneFrequency` minutes.
// It removes unnecessary items from the corpus by calling `Corpus.PruneSequences`.
type CorpusPruner struct {
	// enabled determines if the pruner is enabled
	enabled bool

	// corpus is the corpus to be pruned
	corpus *Corpus

	// logger is used to log when pruning and on error
	logger *logging.Logger

	// ctx is the CorpusPruner's context which can be used to cancel the pruner
	ctx context.Context

	// pruneFrequency determines how often, in minutes, the pruning should occur
	pruneFrequency uint64

	// totalCorpusPruned counts the total number of sequences pruned so far
	totalCorpusPruned int

	// chain is the test chain used during pruning
	chain *chain.TestChain
}

// NewCorpusPruner creates a new CorpusPruner.
func NewCorpusPruner(enabled bool, pruneFrequency uint64, logger *logging.Logger) *CorpusPruner {
	if !enabled {
		return &CorpusPruner{}
	}
	return &CorpusPruner{
		enabled:        enabled,
		pruneFrequency: pruneFrequency,
		logger:         logger,
	}
}

// pruneCorpus is a wrapper around Corpus.PruneSequences that adds timing, logging, and updating totalCorpusPruned.
// It is used by mainLoop.
func (cp *CorpusPruner) pruneCorpus() error {
	start := time.Now() // We'll track how long pruning takes
	n, err := cp.corpus.PruneSequences(cp.ctx, cp.chain)
	// PruneSequences takes a while, so ctx could've finished in the meantime.
	// If it did, we skip the log message.
	if err != nil || utils.CheckContextDone(cp.ctx) {
		return err
	}
	cp.totalCorpusPruned += n
	cp.logger.Info(fmt.Sprintf("Pruned %d values in %v. Total pruned this run: %d", n, time.Since(start), cp.totalCorpusPruned))
	return nil
}

// mainLoop calls pruneCorpus every `pruneFrequency` minutes.
// It runs infinitely until ctx.Done is triggered.
func (cp *CorpusPruner) mainLoop() {
	defer cp.chain.Close()
	ticker := time.NewTicker(time.Duration(cp.pruneFrequency) * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-cp.ctx.Done():
			return
		case <-ticker.C:
			err := cp.pruneCorpus()
			if err != nil {
				cp.logger.Error("Corpus pruner encountered an error", err)
				return
			}
		}
	}
}

// Start takes a context, a corpus to prune, and a base chain in a setup state ready for testing.
// It clones the base chain, then prunes the corpus every `PruneFrequency` minutes.
// This runs until ctx cancels the operation.
// Returns an error if one occurred.
func (cp *CorpusPruner) Start(ctx context.Context, corpus *Corpus, baseTestChain *chain.TestChain) error {
	if !cp.enabled {
		return nil
	}

	// Clone our chain, attaching a tracer.
	clonedChain, err := baseTestChain.Clone(func(initializedChain *chain.TestChain) error {
		initializedChain.AddTracer(coverage.NewCoverageTracer().NativeTracer(), true, false)
		return nil
	})
	if err != nil {
		return err
	}
	cp.chain = clonedChain

	// Write our params to the struct so we don't have to pass them all over the place as function args.
	cp.ctx = ctx
	cp.corpus = corpus

	// Start up the main loop in a goroutine.
	go cp.mainLoop()

	return nil
}
