# Performance benchmarks

## Measured results

Measured on an Apple M4 Max, darwin/arm64, Go 1.26.0, with the default `GOMAXPROCS=16`.
Ten complete campaign samples per configuration and revision were run in alternating baseline /
optimized order. The baseline is commit `87f65e2e9c2b67a95c779742cff5a33e65318600` with the same
benchmark files and embedded Solidity artifact. Comparisons use benchstat
`v0.0.0-20260908200009-22c9c6c9d4da`.

| Workers | Coverage | Baseline µs/tx | Optimized µs/tx | Time comparison       | Allocated bytes | Allocation count |
| ------- | -------- | -------------- | --------------- | --------------------- | --------------- | ---------------- |
| 1       | Enabled  | 101.30         | 96.81           | -4.43%, p=0.002       | -7.64%          | -10.72%          |
| 1       | Disabled | 79.19          | 76.79           | -3.04%, p<0.001       | -4.87%          | -8.93%           |
| 4       | Enabled  | 32.29          | 31.62           | Inconclusive, p=0.280 | -7.95%          | -11.02%          |
| 4       | Disabled | 24.88          | 24.38           | Inconclusive, p=0.075 | -5.57%          | -9.48%           |

The allocation reductions are statistically significant in all four configurations (p<0.001).
Neither four-worker configuration establishes a timing improvement. Average gas per fuzz call was
about 44,400 on both revisions. The four-worker configuration without coverage used 0.11% less gas
per call (p=0.002); the other gas comparisons are inconclusive. This checks for gross differences
in executed work, not identical call sequences. The benchmark uses 16,384 calls to amortize startup
and final shrinking while still measuring them.

Raw campaign samples: [baseline](campaign-baseline.txt), [optimized](campaign-optimized.txt), and
the complete [benchstat comparison](campaign-comparison.txt).

Component benchmarks use ten 100 ms samples per revision:

| Component                       | Median time change | Allocation count, before → after |
| ------------------------------- | ------------------ | -------------------------------- |
| Nested coverage frames          | -78.90%            | 57 → 8                           |
| Provider-style EVM call         | -11.32%            | 93 → 82                          |
| Integer generation, 8 seeds     | -50.17%            | 28 → 13                          |
| Integer generation, 256 seeds   | -23.82%            | 27 → 14                          |
| Integer generation, 4,096 seeds | -23.24%            | 28 → 14                          |
| In-range integer constraint     | -64.70%            | 4 → 2                            |
| Overflow constraint             | -38.48%            | 10 → 4                           |
| Underflow constraint            | -38.59%            | 10 → 4                           |

All listed component timing differences have p<0.001. Raw component samples:
[baseline](components-baseline.txt), [optimized](components-optimized.txt), and
[benchstat comparison](components-comparison.txt). These isolated improvements are not whole-fuzzer
speedup percentages.

Seed-selection and error-decoding benchmarks use ten 100 ms samples per revision. Selected
results are below; [baseline samples](selection-errors-baseline.txt),
[optimized samples](selection-errors-optimized.txt), and the
[complete comparison](selection-errors-comparison.txt) also include random fallback and
supplied-input mutation. The unchanged random-address path is a control, not an optimization.

| Component                         | Median time change | Allocation count, before → after |
| --------------------------------- | ------------------ | -------------------------------- |
| Byte seed selection, 256 seeds    | -64.26%            | 2 → 1                            |
| String seed selection, 256 seeds  | -63.04%            | 1 → 0                            |
| Address seed selection, 256 seeds | -64.27%            | 1 → 0                            |
| Recognized Solidity panic         | -96.61%            | 18 → 2                           |
| Recognized Solidity error string  | -79.59%            | 26 → 12                          |
| Unrecognized panic/error selector | About -99.4%       | 14 → 0                           |

ABI packing uses ten alternating 300 ms samples per revision:

| Arguments                  | Median time change | Allocation count, before → after |
| -------------------------- | ------------------ | -------------------------------- |
| Empty                      | -18.52%            | 1 → 1                            |
| Integer                    | -10.62%            | 5 → 4                            |
| Dynamic bytes/string/array | -9.81%             | 32 → 30                          |

All timing differences listed in these two tables have p<0.001. Dynamic packing allocates 18.21%
fewer bytes. See the [baseline samples](abi-pack-baseline.txt),
[optimized samples](abi-pack-optimized.txt), and [comparison](abi-pack-comparison.txt).

## Workload

`BenchmarkFuzzer` runs real in-memory fuzzing campaigns with one and four workers, with coverage
enabled and disabled. Each operation targets 16,384 fuzz calls. It includes worker creation and
reset, ABI value generation, corpus mutation, transaction execution, branch coverage, assertion
testing, property calls, optimization calls, and final optimization shrinking. Workers use separate
chains and share the normal corpus and providers.

The Solidity fixture performs storage writes, hashing in a loop, nested external calls, caught
reverts, and dynamic byte/string argument and return-value handling. Its property and assertion hold
for every input; its optimization target varies with state.
`TestBenchmarkWorkload` verifies actual work, coverage collection, test registration, and activity in
each worker. The benchmark stops through an additional post-call hook, avoiding the production
transaction limit's three-second polling interval. It reports actual `tx/s`, `ns/tx`, and average
`gas/tx`, including workers finishing in-flight calls when the target count is reached. Provider calls and shrinking
are included in elapsed time, but are not counted as generated fuzz calls.

Compilation, artifact parsing, and initial fuzzer construction are outside the timer. Campaign
startup, shutdown, and shrinking are inside it. The fixture is embedded, so benchmark execution
requires neither Solidity tools nor network access. Disk persistence, RPC forking, report rendering,
and external compilation are outside this workload. The worker PRNG has a fixed seed, but Go map
iteration, corpus selection, scheduling, and shrinking still introduce variation. Use repeated
measurements rather than comparing individual runs.

Additional native benchmarks isolate nested coverage collection, provider-style EVM calls, integer
generation at three seed-set sizes, integer wraparound, byte/string/address seed selection, ABI
packing, and standard Solidity error decoding.

## Run and compare

From the repository root:

```sh
go test ./fuzzing -run '^$' -bench '^BenchmarkFuzzer$' \
  -benchmem -benchtime=1x -count=10 > campaign.txt

go test -p 1 ./chain ./fuzzing/coverage ./fuzzing/valuegeneration ./utils \
  -run '^$' \
  -bench 'Benchmark(ChainCall|CoverageNestedFrames|IntegerGeneration|ConstrainInteger)$' \
  -benchmem -benchtime=100ms -count=10 > components.txt

go test -p 1 ./fuzzing/valuegeneration ./compilation/abiutils -run '^$' \
  -bench 'Benchmark(SeedSelection|SolidityErrors)$' \
  -benchmem -benchtime=100ms -count=10 > selection-errors.txt

go test ./fuzzing/calls -run '^$' -bench '^BenchmarkABIPack$' \
  -benchmem -benchtime=300ms -count=10 > abi-pack.txt
```

Use the same machine, Go version, `GOMAXPROCS`, fixture, benchmark code, and benchmark flags for both
revisions. Avoid concurrent tests, builds, or benchmarks while collecting timing samples. Alternate
baseline and optimized runs to reduce order effects. The campaign's `ns/op` is per campaign;
`ns/tx` is normalized by actual generated calls. `B/op` and `allocs/op` are per campaign too.

The baseline for this change is `87f65e2e9c2b67a95c779742cff5a33e65318600`. To benchmark that revision,
create a detached worktree and copy only these newly added benchmark/test fixtures into it:

- `chain/call_benchmark_test.go`
- `fuzzing/fuzzer_benchmark_test.go`
- `fuzzing/coverage/coverage_tracer_test.go`
- `fuzzing/valuegeneration/integer_benchmark_test.go`
- `fuzzing/valuegeneration/seed_selection_test.go`
- `fuzzing/calls/abi_pack_test.go`
- `compilation/abiutils/solidity_errors_test.go`
- `utils/integer_utils_test.go`
- `fuzzing/testdata/benchmark.sol`
- `fuzzing/testdata/benchmark.json`

Compare native output using [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat):

```sh
benchstat baseline.txt optimized.txt
```

For profiles, collect a separate run; do not use its timings for the comparison:

```sh
go test ./fuzzing -run '^$' -bench '^BenchmarkFuzzer/workers=1/coverage=true$' \
  -benchtime=3x -cpuprofile=cpu.out -memprofile=heap.out
go tool pprof -top cpu.out
go tool pprof -alloc_space -top heap.out
```

## Changes

- Record coverage directly in one transaction map. Previously each call frame allocated a map and
  merged it into its parent, even though coverage retains reverted branches. Preserve separate
  transaction results, marker hit counts, code identities, and address grouping.
- Reuse the coverage frame slice and store frames and lookup hashes by value, avoiding per-frame
  heap objects. Resolve bytecode identity once per frame and cache only runtime hashes.
- Reuse the chain's call tracer router when a call supplies no additional tracers. Preserve the
  existing router composition when additional tracers are present.
- Build integer seed slices with room for the boundary values, avoiding repeated slice copies and
  growth in generation and shrinking.
- Reuse the generator's private integer accumulator during arithmetic mutations. Caller inputs and
  seed integers remain unchanged.
- Calculate powers of two using shifts when constructing integer bounds.
- Return an independent copy immediately for already bounded integers. For overflow and underflow,
  wrap using Euclidean modulo instead of multiple distance, quotient, and correction temporaries.
- Select a uniform random byte/string/address seed by walking to a sampled map index. Avoid
  materializing entire seed lists, and skip iteration for random fallback or supplied-input mutation.
  Copy mutable byte seeds before mutation and preserve the generator's random-number draws.
- Reuse spare capacity in the independently owned ABI argument buffer when prepending the
  selector. Otherwise allocate the complete call data once. Retain ABI validation and independent
  output, including empty or custom selectors.
- Use standard Solidity error selectors directly. Decode panic integers from their fixed 32-byte
  payload and reuse a private, read-only string ABI. Preserve malformed-input rejection, wrapped
  revert errors, and legacy invalid-opcode behavior.

## Remaining opportunities

The final [CPU profile](cpu-profile.txt) attributes about 22% of sampled CPU cumulatively to opcode
tracer routing. Flattening callback dispatch may help, but externally supplied tracers expose mutable hooks;
caching callbacks must preserve those semantics. Deployment tracing currently visits every opcode
to recognize `SELFDESTRUCT`; moving it to the EVM enter callback would change observation timing
because that callback runs after the state transition. Neither change is made here.

The final [allocation profile](heap-profile.txt) also highlights geth access lists and storage
journals. Reusing EVM instances and jump-destination analysis across transactions may help, but requires accounting for snapshots,
cheatcode changes to execution contexts, cancellation, and geth's transaction-local state.

Console cheatcode ABI construction is repeated during worker reset, including hundreds of method
signatures. This is also tracked in [issue #385](https://github.com/crytic/medusa/issues/385). Sharing
definitions would need to preserve the independently mutable ABI exposed by each chain. Corpus
sequence cloning also serializes and deserializes ABI arguments; a direct clone must preserve type
validation, tuple/array handling, and independent mutable values.

These are follow-up candidates, not measured improvements delivered by this change. The synthetic
benchmark cannot establish that every contract, worker count, or forked campaign gets faster.

## Validation

The regression tests cover nested and reverted coverage, repeated contract addresses, init versus
runtime code identity, independence of transaction results, callback ordering and isolation,
storage rollback, integer input/seed preservation, integer bounds, random fallback, empty seeds,
byte ownership, ABI packing errors, and malformed Solidity error payloads. Integer wraparound is also
checked exhaustively on small intervals and with a native Go fuzz target on large values.

```sh
go test ./... -skip '^TestDisabledColors$' -count=1
go test -race ./utils ./fuzzing/valuegeneration ./chain ./fuzzing/coverage \
  ./fuzzing/calls ./compilation/abiutils
go test ./utils -run '^$' -fuzz '^FuzzConstrainIntegerToBounds$' \
  -fuzztime=30s -parallel=4
go test ./compilation/abiutils -run '^$' -fuzz '^FuzzSolidityErrorDecoding$' \
  -fuzztime=30s -parallel=4
golangci-lint run --timeout 5m
```

The unchanged logging test `TestDisabledColors` fails on the baseline and has an existing fix in
[PR #822](https://github.com/crytic/medusa/pull/822). The full suite passes when only that test is
excluded. The parallel fuzzer workload also reproduces existing provider, corpus, and worker
coordination data races on both revisions; race-detector failures are tracked in
[issue #269](https://github.com/crytic/medusa/issues/269). Race checks for the changed chain,
coverage, value-generation, call-packing, error-decoding, and integer utility packages pass. These
existing bugs are not patched as part of the performance changes.

Two 30-second native fuzz runs completed 1,523,278 integer-bound executions and 2,971,134
error-decoding executions without a failure. Linux/amd64 and Windows/amd64 builds also passed with `CGO_ENABLED=0`; these are compilation checks, not runtime
tests on those operating systems:

```sh
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...
```

Mutation checks deliberately changed the integer wrap width, replaced the revert marker with a
return marker, aliased a returned byte seed, and substituted an incorrect panic selector. The new
regression tests rejected all four changes; the working implementations were restored and retested.

## Regenerate the fixture

The checked-in artifact was compiled with solc `0.8.25+commit.b61c2a91`, without optimization:

```sh
solc fuzzing/testdata/benchmark.sol --combined-json abi,bin,bin-runtime \
  > fuzzing/testdata/benchmark.json
dprint fmt fuzzing/testdata/benchmark.json
```

Changing the compiler, source, or optimization flags changes the measured workload. Regenerate and
use the same artifact for both sides of any subsequent comparison.
