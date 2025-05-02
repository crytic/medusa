package fuzzing

import (
	"fmt"
	"time"

	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/fuzzing/coverage"
	"github.com/crytic/medusa/utils"
)

type CorpusPruner struct {
	enabled           bool
	fuzzer            *Fuzzer
	totalCorpusPruned int
	chain             *chain.TestChain
}

func NewCorpusPruner(enabled bool, fuzzer *Fuzzer) *CorpusPruner {
	if !enabled {
		return &CorpusPruner{}
	}
	return &CorpusPruner{
		enabled: enabled,
		fuzzer:  fuzzer,
	}
}

func (cp *CorpusPruner) pruneCorpus() error {
	start := time.Now() // We'll track how long pruning takes
	n, err := cp.fuzzer.corpus.PruneSequences(cp.fuzzer.ctx, cp.chain)
	if err != nil || utils.CheckContextDone(cp.fuzzer.ctx) {
		return err
	}
	cp.totalCorpusPruned += n
	cp.fuzzer.logger.Info(fmt.Sprintf("Pruned %d values in %v. Total pruned this run: %d", n, time.Since(start), cp.totalCorpusPruned))
	return nil
}

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
				cp.fuzzer.Terminate()
				return
			}
		}
	}
}

// run takes a base Chain in a setup state ready for testing, clones it, and TODO
// This runs until Fuzzer.ctx cancels the operation.
// Returns a boolean indicating whether Fuzzer.ctx or Fuzzer.emergencyCtx has indicated we cancel the operation, and an
// error if one occurred.
func (cp *CorpusPruner) Start(baseTestChain *chain.TestChain) error {
	if !cp.enabled {
		return nil
	}

	var err error
	// Clone our chain, attaching our necessary components for fuzzing post-genesis, prior to all blocks being copied.
	// This means any tracers added or events subscribed to within this inner function are done so prior to chain
	// setup (initial contract deployments), so data regarding that can be tracked as well.
	cp.chain, err = baseTestChain.Clone(func(initializedChain *chain.TestChain) error {
		initializedChain.AddTracer(coverage.NewCoverageTracer().NativeTracer(), true, false)
		return nil
	})

	// If we encountered an error during cloning, return it.
	if err != nil {
		return err
	}

	go cp.mainLoop()

	return nil
}
