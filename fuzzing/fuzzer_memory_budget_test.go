//go:build memprofile

package fuzzing

import (
	"context"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/crytic/medusa/fuzzing/calls"
	"github.com/crytic/medusa/fuzzing/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests are excluded from the default test run because they take minutes. Run them with:
//
//	go test -tags memprofile -run TestMemoryBudget -timeout 30m -v ./fuzzing/

const (
	// memoryBudgetSampleInterval is how often the live heap is sampled during a campaign.
	memoryBudgetSampleInterval = 200 * time.Millisecond

	// memoryBudgetCampaignTimeout is how long, in seconds, the plain campaign runs. It must exceed one minute so the
	// corpus pruner (which ticks in minutes) runs at least once.
	memoryBudgetCampaignTimeout = 150

	// memoryBudgetCampaignLimit is the live heap, in bytes, the plain campaign may not exceed.
	memoryBudgetCampaignLimit = 64 * 1024 * 1024

	// memoryBudgetPrunerTimeout is how long, in seconds, the injected corpus campaign runs. It leaves time after the
	// first pruner tick for a full pass over the injected corpus.
	memoryBudgetPrunerTimeout = 100

	// memoryBudgetInjectedSequences is how many full-length call sequences are injected into the corpus.
	memoryBudgetInjectedSequences = 300

	// memoryBudgetPrunerLimit is the live heap, in bytes, the injected corpus campaign may not exceed. A pruner pass
	// over the injected corpus peaked below 190 MB locally, and near 310 MB with the pruner coverage cleanup from #812
	// removed.
	memoryBudgetPrunerLimit = 240 * 1024 * 1024
)

// heapSampler records the live heap of the current process at a fixed interval.
type heapSampler struct {
	// start is the time sampling began.
	start time.Time

	// first is the first live heap sample, in bytes.
	first uint64

	// last is the most recent live heap sample, in bytes.
	last uint64

	// peak is the largest live heap sample observed, in bytes.
	peak uint64

	// lastAt is the time the most recent sample was taken.
	lastAt time.Time

	// cancel stops the sampling goroutine.
	cancel context.CancelFunc

	// done is closed once the sampling goroutine exits.
	done chan struct{}

	// lock guards the sample fields above.
	lock sync.Mutex
}

// startHeapSampler begins sampling the live heap every interval until stop is called.
func startHeapSampler(interval time.Duration) *heapSampler {
	ctx, cancel := context.WithCancel(context.Background())
	s := &heapSampler{start: time.Now(), cancel: cancel, done: make(chan struct{})}
	s.sample()

	go func() {
		defer close(s.done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.sample()
			}
		}
	}()
	return s
}

// sample forces a garbage collection so HeapAlloc reflects reachable memory rather than garbage awaiting
// collection, then records it.
func (s *heapSampler) sample() {
	runtime.GC()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	s.lock.Lock()
	defer s.lock.Unlock()
	if s.first == 0 {
		s.first = stats.HeapAlloc
	}
	s.last = stats.HeapAlloc
	s.lastAt = time.Now()
	if stats.HeapAlloc > s.peak {
		s.peak = stats.HeapAlloc
	}
}

// stop takes a final sample and stops the sampling goroutine.
func (s *heapSampler) stop() {
	s.cancel()
	<-s.done
	s.sample()
}

// report logs the first, last, and peak live heap along with the growth rate over the run.
func (s *heapSampler) report(t *testing.T) {
	s.lock.Lock()
	defer s.lock.Unlock()
	const mb = 1024 * 1024
	minutes := s.lastAt.Sub(s.start).Minutes()
	growth := (float64(s.last) - float64(s.first)) / mb / minutes
	t.Logf("live heap: first %.1f MB, last %.1f MB, peak %.1f MB, growth %.2f MB/min over %.1f min",
		float64(s.first)/mb, float64(s.last)/mb, float64(s.peak)/mb, growth, minutes)
}

// configureMemoryBudgetCampaign applies the settings shared by every memory budget test: coverage and the corpus
// pruner on, no test limit, and failing tests that do not stop the campaign early.
func configureMemoryBudgetCampaign(projectConfig *config.ProjectConfig, timeout int) {
	projectConfig.Fuzzing.TargetContracts = []string{"GenerateAllTypes"}
	projectConfig.Fuzzing.Workers = 2
	projectConfig.Fuzzing.Timeout = timeout
	projectConfig.Fuzzing.TestLimit = 0
	projectConfig.Fuzzing.CoverageEnabled = true
	projectConfig.Fuzzing.CoverageFormats = nil
	projectConfig.Fuzzing.CorpusDirectory = ""
	projectConfig.Fuzzing.PruneFrequency = 1
	projectConfig.Fuzzing.Testing.StopOnFailedTest = false
	projectConfig.Fuzzing.Testing.StopOnFailedContractMatching = false
	projectConfig.Slither.UseSlither = false
}

// TestMemoryBudgetCampaign runs a fuzzing campaign long enough for the corpus pruner to run and fails if the live
// heap exceeds a fixed budget. It is coarse and only catches catastrophic growth.
func TestMemoryBudgetCampaign(t *testing.T) {
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/value_generation/generate_all_types.sol",
		configUpdates: func(projectConfig *config.ProjectConfig) {
			configureMemoryBudgetCampaign(projectConfig, memoryBudgetCampaignTimeout)
		},
		method: func(f *fuzzerTestContext) {
			sampler := startHeapSampler(memoryBudgetSampleInterval)
			err := f.fuzzer.Start()
			sampler.stop()
			require.NoError(f.t, err)

			sampler.report(f.t)
			assert.LessOrEqual(f.t, sampler.peak, uint64(memoryBudgetCampaignLimit), "live heap exceeded the memory budget")
		},
	})
}

// TestMemoryBudgetPrunerCorpus injects a large corpus of full-length call sequences so the corpus pruner replays
// thousands of calls in a single pass, then fails if the live heap exceeds a fixed budget. Simple contracts never
// grow a corpus this large on their own, so without injection the pruner coverage leak fixed in #812 never shows.
func TestMemoryBudgetPrunerCorpus(t *testing.T) {
	runFuzzerTest(t, &fuzzerSolcFileTest{
		filePath: "testdata/contracts/value_generation/generate_all_types.sol",
		configUpdates: func(projectConfig *config.ProjectConfig) {
			configureMemoryBudgetCampaign(projectConfig, memoryBudgetPrunerTimeout)
		},
		method: func(f *fuzzerTestContext) {
			// Copy every full-length sequence a worker executes into the corpus until enough have been injected.
			// Chain references are dropped so the injected copies do not keep the worker's blocks alive.
			var injected atomic.Int64
			f.fuzzer.Hooks.CallSequenceTestFuncs = append(f.fuzzer.Hooks.CallSequenceTestFuncs, func(worker *FuzzerWorker, callSequence calls.CallSequence) ([]ShrinkCallSequenceRequest, error) {
				if len(callSequence) < worker.fuzzer.config.Fuzzing.CallSequenceLength {
					return nil, nil
				}
				if injected.Add(1) > memoryBudgetInjectedSequences {
					return nil, nil
				}
				clone, err := callSequence.Clone()
				if err != nil {
					return nil, err
				}
				for _, element := range clone {
					element.ChainReference = nil
					element.ExecutionTrace = nil
				}
				return nil, worker.fuzzer.corpus.MarkCallSequenceForMutation(clone, big.NewInt(1))
			})

			sampler := startHeapSampler(memoryBudgetSampleInterval)
			err := f.fuzzer.Start()
			sampler.stop()
			require.NoError(f.t, err)

			sampler.report(f.t)
			f.t.Logf("injected %d sequences, %d remain after pruning", min(injected.Load(), memoryBudgetInjectedSequences), f.fuzzer.corpus.ActiveMutableSequenceCount())

			// If the pruner never ran, nothing was removed and the budget below proves nothing.
			require.GreaterOrEqual(f.t, injected.Load(), int64(memoryBudgetInjectedSequences), "corpus injection did not complete")
			require.Less(f.t, f.fuzzer.corpus.ActiveMutableSequenceCount(), memoryBudgetInjectedSequences, "corpus pruner did not run")
			assert.LessOrEqual(f.t, sampler.peak, uint64(memoryBudgetPrunerLimit), "live heap exceeded the memory budget")
		},
	})
}
