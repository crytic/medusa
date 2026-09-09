package fuzzing

import (
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"

	"github.com/crytic/medusa/compilation"
	compilationTypes "github.com/crytic/medusa/compilation/types"
	"github.com/crytic/medusa/fuzzing/calls"
	"github.com/crytic/medusa/fuzzing/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/benchmark.json
var benchmarkArtifact []byte

//go:embed testdata/benchmark.sol
var benchmarkSource []byte

func benchmarkCompilation(tb testing.TB) compilationTypes.Compilation {
	tb.Helper()
	var artifact struct {
		Contracts map[string]struct {
			ABI     json.RawMessage `json:"abi"`
			Bin     string          `json:"bin"`
			Runtime string          `json:"bin-runtime"`
		} `json:"contracts"`
	}
	require.NoError(tb, json.Unmarshal(benchmarkArtifact, &artifact))
	compiled := artifact.Contracts["fuzzing/testdata/benchmark.sol:Benchmark"]
	contractABI, err := compilationTypes.ParseABIFromInterface(string(compiled.ABI))
	require.NoError(tb, err)
	initCode, err := hex.DecodeString(compiled.Bin)
	require.NoError(tb, err)
	runtimeCode, err := hex.DecodeString(compiled.Runtime)
	require.NoError(tb, err)
	result := compilationTypes.NewCompilation()
	result.SourceCode["benchmark.sol"] = benchmarkSource
	result.SourcePathToArtifact["benchmark.sol"] = compilationTypes.SourceArtifact{
		Contracts: map[string]compilationTypes.CompiledContract{
			"Benchmark": {Abi: *contractABI, InitBytecode: initCode, RuntimeBytecode: runtimeCode},
		},
	}
	return *result
}

func newBenchmarkFuzzer(tb testing.TB, workers int, coverage bool) *Fuzzer {
	tb.Helper()
	cfg, err := config.GetDefaultProjectConfig("")
	require.NoError(tb, err)
	cfg.Slither.UseSlither = false
	cfg.Logging.Level = zerolog.Disabled
	cfg.Fuzzing.Workers = workers
	cfg.Fuzzing.TargetContracts = []string{"Benchmark"}
	cfg.Fuzzing.CallSequenceLength = 32
	cfg.Fuzzing.ShrinkLimit = 128
	cfg.Fuzzing.CoverageEnabled = coverage
	cfg.Fuzzing.CoverageFormats = nil
	cfg.Fuzzing.Testing.TestViewMethods = false
	f, err := NewFuzzer(*cfg)
	require.NoError(tb, err)
	f.config.Compilation, err = compilation.NewCompilationConfig("solc")
	require.NoError(tb, err)
	f.AddCompilationTargets([]compilationTypes.Compilation{benchmarkCompilation(tb)})
	f.Events.FuzzerStarting.Subscribe(func(FuzzerStartingEvent) error {
		f.randomProvider = rand.New(rand.NewSource(1))
		return nil
	})
	return f
}

func limitBenchmarkCalls(f *Fuzzer, limit int64) *atomic.Int64 {
	count := new(atomic.Int64)
	f.Hooks.CallSequenceTestFuncs = append(f.Hooks.CallSequenceTestFuncs,
		func(_ *FuzzerWorker, _ calls.CallSequence) ([]ShrinkCallSequenceRequest, error) {
			if count.Add(1) >= limit {
				f.Stop()
			}
			return nil, nil
		})
	return count
}

// BenchmarkFuzzer measures complete in-memory campaigns, including worker startup/reset,
// generation/mutation, transactions, coverage/corpus updates, all providers and final shrinking.
// Compilation and fixture loading are outside the timer. One operation targets 16384 fuzz calls;
// tx/s and ns/tx use the actual count, including workers finishing in-flight calls on stop.
func BenchmarkFuzzer(b *testing.B) {
	for _, workers := range []int{1, 4} {
		for _, coverage := range []bool{true, false} {
			b.Run(fmt.Sprintf("workers=%d/coverage=%t", workers, coverage), func(b *testing.B) {
				var total int64
				var totalGas float64
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					f := newBenchmarkFuzzer(b, workers, coverage)
					count := limitBenchmarkCalls(f, 16384)
					b.StartTimer()
					err := f.Start()
					b.StopTimer()
					require.NoError(b, err)
					require.GreaterOrEqual(b, count.Load(), int64(16384))
					total += count.Load()
					gasUsed, _ := f.metrics.GasUsed().Float64()
					totalGas += gasUsed
				}
				b.ReportMetric(float64(total)/b.Elapsed().Seconds(), "tx/s")
				b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(total), "ns/tx")
				b.ReportMetric(totalGas/float64(total), "gas/tx")
			})
		}
	}
}

func TestBenchmarkWorkload(t *testing.T) {
	f := newBenchmarkFuzzer(t, 4, true)
	count := limitBenchmarkCalls(f, 512)
	var integerCalls, payloadCalls atomic.Int64
	f.Hooks.CallSequenceTestFuncs = append(f.Hooks.CallSequenceTestFuncs,
		func(_ *FuzzerWorker, sequence calls.CallSequence) ([]ShrinkCallSequenceRequest, error) {
			data := sequence[len(sequence)-1].Call.DataAbiValues
			if data != nil {
				switch data.Method.RawName {
				case "step":
					integerCalls.Add(1)
				case "payload":
					payloadCalls.Add(1)
				}
			}
			return nil, nil
		})
	require.NoError(t, f.Start())
	require.Positive(t, integerCalls.Load())
	require.Positive(t, payloadCalls.Load())
	require.GreaterOrEqual(t, count.Load(), int64(512))
	require.Equal(t, count.Load(), f.metrics.CallsTested().Int64())
	require.Greater(t, f.corpus.CoverageMaps().BranchesHit(), uint64(10))
	require.Empty(t, f.TestCasesWithStatus(TestCaseStatusFailed))
	require.Len(t, f.TestCases(), 4)
	for _, metrics := range f.metrics.workerMetrics {
		require.Positive(t, metrics.callsTested.Int64())
	}
}
