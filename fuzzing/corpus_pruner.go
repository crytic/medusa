package fuzzing

import (
	"fmt"
	"time"

	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/fuzzing/coverage"
	"github.com/crytic/medusa/utils"
)

// CorpusPruner is a job that runs every `PruneFrequency` minutes.
// It removes unnecessary items from the corpus by calling `Corpus.PruneSequences`.
type CorpusPruner struct {
	// enabled determines if the pruner is enabled
	enabled bool

	// fuzzer describes the Fuzzer instance which this pruner belongs to
	fuzzer *Fuzzer

	// totalCorpusPruned counts the total number of sequences pruned so far
	totalCorpusPruned int

	// chain is the test chain used during pruning
	chain *chain.TestChain
}

// NewCorpusPruner creates a new CorpusPruner.
func NewCorpusPruner(enabled bool, fuzzer *Fuzzer) *CorpusPruner {
	if !enabled {
		return &CorpusPruner{}
	}
	return &CorpusPruner{
		enabled: enabled,
		fuzzer:  fuzzer,
	}
}

// pruneCorpus is a wrapper around Corpus.PruneSequences that adds timing, logging, and updating totalCorpusPruned.
// It is used by mainLoop.
func (cp *CorpusPruner) pruneCorpus() error {
	start := time.Now() // We'll track how long pruning takes
	n, err := cp.fuzzer.corpus.PruneSequences(cp.fuzzer.ctx, cp.chain)
	// PruneSequences takes a while, so ctx could've finished in the meantime.
	// If it did, we skip the log message.
	if err != nil || utils.CheckContextDone(cp.fuzzer.ctx) {
		return err
	}
	cp.totalCorpusPruned += n
	cp.fuzzer.logger.Info(fmt.Sprintf("Pruned %d values in %v. Total pruned this run: %d", n, time.Since(start), cp.totalCorpusPruned))
	return nil
}

// mainLoop calls pruneCorpus every `PruneFrequency` minutes.
// It runs infinitely until fuzzer.ctx.Done is triggered.
func (cp *CorpusPruner) mainLoop() {
	defer cp.chain.Close()
	ticker := time.NewTicker(time.Duration(cp.fuzzer.Config().Fuzzing.PruneFrequency) * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-cp.fuzzer.ctx.Done():
			return
		case <-ticker.C:
			err := cp.pruneCorpus()
			if err != nil {
				cp.fuzzer.logger.Error("Corpus pruner encountered an error", err)
				cp.fuzzer.Terminate() // We kill the whole thing if the pruner errors
				return
			}
		}
	}
}

// Start takes a base Chain in a setup state ready for testing, clones it, and prunes the corpus every `PruneFrequency` minutes.
// This runs until Fuzzer.ctx cancels the operation.
// Returns an error if one occurred.
func (cp *CorpusPruner) Start(baseTestChain *chain.TestChain) error {
	if !cp.enabled {
		return nil
	}

	var err error

	// Clone our chain, attaching a tracer.
	cp.chain, err = baseTestChain.Clone(func(initializedChain *chain.TestChain) error {
		initializedChain.AddTracer(coverage.NewCoverageTracer().NativeTracer(), true, false)
		return nil
	})
	if err != nil {
		return err
	}

	// Start up the main loop in a goroutine.
	go cp.mainLoop()

	return nil
}
