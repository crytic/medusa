# Fuzzing Engine (`/fuzzing`)

The [`/fuzzing`](../../fuzzing) package is the heart of medusa. This page explains its subsystems and how they
interact during a campaign. For the step-by-step campaign flow, see
[../workflows/fuzzing-lifecycle.md](../workflows/fuzzing-lifecycle.md).

## Fuzzer (`fuzzer.go`)

`Fuzzer` is the top-level orchestrator. It:

- Holds the `ProjectConfig`, the list of compilations, and the derived `contractDefinitions`.
- Owns the shared `baseValueSet` (seed values for fuzzing), the `corpus`, `metrics`, and optional
  `revertReporter` and `corpusPruner`.
- Manages the worker pool and two contexts: a normal `ctx` (best-effort cancellation) and an `emergencyCtx`
  (guaranteed termination on SIGINT/errors).
- Registers the built-in test case providers on construction.

Entry points: `NewFuzzer(config)` and `Fuzzer.Start()`. See [`fuzzing/fuzzer.go`](../../fuzzing/fuzzer.go).

## Worker (`fuzzer_worker.go`)

`FuzzerWorker` is a single fuzzing thread. Each worker owns:

- Its own `chain.TestChain` (isolated EVM) and a `coverageTracer`.
- `testingBaseBlockIndex` — the block after all target contracts are deployed; the worker **reverts to this block
  after every sequence** to reset state cheaply.
- Classified method lists: `stateChangingMethods`, `pureMethods` (view/pure), and `fallbackMethods`.
- A `sequenceGenerator`, a `shrinkingValueMutator`, and a worker-local `valueSet` derived from the fuzzer's base set
  plus runtime values.

The worker runs the inner fuzzing loop and processes any pending `shrinkCallSequenceRequests` before generating the
next sequence. See [`fuzzing/fuzzer_worker.go`](../../fuzzing/fuzzer_worker.go).

## Call sequences (`/fuzzing/calls`)

- **`CallSequence`** — an ordered array of `CallSequenceElement`s executed against the test chain while maintaining
  state (until the sequence ends and state is reset).
- **`CallSequenceElement` / `CallMessage`** — a single call: target address, method, ABI-encoded arguments, block
  number/timestamp delay, gas limit, and sender.

Execution helpers live in [`fuzzing/calls/call_sequence_execution.go`](../../fuzzing/calls/call_sequence_execution.go).
The maximum sequence length is `fuzzing.callSequenceLength` (default 100).

## Sequence generation (`fuzzer_worker_sequence_generator.go`)

`CallSequenceGenerator` builds each new sequence either from scratch or by mutating corpus entries, chosen by a
**weighted random selector**. Configurable strategies (see `CallSequenceGeneratorConfig`) include:

- New entirely-random sequence (`NewSequenceProbability`).
- Take corpus **head** and append new calls; take corpus **tail** and prepend new calls.
- **Splice** two corpus sequences; **interleave** calls from two corpus sequences.
- Mutated variants of the above (with value mutations applied to arguments).

This is the mechanism that turns coverage feedback into deeper exploration. Reference:
[`fuzzing/fuzzer_worker_sequence_generator.go`](../../fuzzing/fuzzer_worker_sequence_generator.go).

## Value generation (`/fuzzing/valuegeneration`)

Generates and mutates ABI argument values.

- `generator_random.go` — random value generation for each ABI type.
- `generator_mutational.go` — mutates existing values (bit flips, boundary values, value-set values such as
  AST/Slither constants and runtime return values).
- `mutator_shrinking.go` — value mutations used during shrinking (see below).
- `value_set.go` / `value_set_from_ast.go` / `value_set_from_slither.go` — the seed **value set**, populated from
  compilation AST constants and Slither-extracted constants (recent history: commit _"skip invalid Slither
  constants during seeding"_ touched `value_set_from_slither.go`).
- `abi_values.go` — encoding/decoding and value helpers for ABI types.

## Corpus (`/fuzzing/corpus`)

The corpus persists **coverage-increasing** call sequences so they can be replayed and mutated in future runs.

- Stored on disk under the configured `corpus/` directory (in-memory only if `corpusDirectory` is empty).
- A background **pruner** (`corpus_pruner`, controlled by `fuzzing.pruneFrequency`, default 5 minutes) removes
  redundant sequences to bound size and memory. Pruning only runs when coverage is enabled.
- `corpus_cleaner.go` supports the `medusa corpus clean` CLI command, which removes invalid/unreplayable sequences.

Reference: [`fuzzing/corpus/corpus.go`](../../fuzzing/corpus/corpus.go).

## Coverage (`/fuzzing/coverage`)

- `coverage_tracer.go` — an EVM tracer that records executed code locations (jumps, returns, reverts, contract
  entrance) per contract.
- `coverage_maps.go` — the coverage data structures used to decide whether a sequence increased coverage.
- `source_analysis.go` + `report_generation.go` + `report_template.gohtml` — map bytecode coverage back to source
  and render HTML/LCOV reports. Report formats are set via `fuzzing.coverageFormats` (default `html`, `lcov`);
  `fuzzing.coverageExclusions` supports glob patterns.

## Test case providers (`test_case_*`)

Three providers run concurrently and register `CallSequenceTestFunc` hooks:

1. **Assertion** (`test_case_assertion_provider.go`) — detects EVM panic conditions per method (e.g. `assert()`,
   overflow/underflow, divide-by-zero, out-of-bounds). Which panic codes fail is set via
   `PanicCodeConfig` under `testing.assertionTesting`.
2. **Property** (`test_case_property_provider.go`) — calls no-argument `bool`-returning functions matching
   configured prefixes (default `property_`), invoked read-only; a `false` return (or revert) fails the test.
   (`view`/`pure` is convention — state mutability is not enforced.)
3. **Optimization** (`test_case_optimization_provider.go`) — tracks a maximization target and shrinks sequences to
   find minimal paths that maximize it (default prefix `optimize_`).

Each `TestCase` (`test_case.go`) tracks Status (`NOT STARTED`, `RUNNING`, `PASSED`, `FAILED`), name, failing call
sequence, and a result message. See [../testing.md](../testing.md) for how to write these tests.

## Shrinking (`fuzzer_worker_shrinking.go`)

When a test fails, a `ShrinkCallSequenceRequest` is queued. The worker iteratively removes/mutates calls and values
while a `VerifierFunction` confirms the test still fails, producing a **minimal reproduction**. The number of
shrinking iterations is bounded by `fuzzing.shrinkLimit` (default 5000). Value mutations used here come from
`valuegeneration/mutator_shrinking.go`.

## Execution tracing & reverts

- [`/fuzzing/executiontracer`](../../fuzzing/executiontracer) — produces human-readable execution traces for failing
  sequences (used in reporting).
- [`/fuzzing/reverts`](../../fuzzing/reverts) — the `RevertReporter` collects per-function revert metrics when
  `fuzzing.revertReporterEnabled` is set.

## Change-oriented notes

- Most engine changes have tests in [`fuzzing/fuzzer_test.go`](../../fuzzing/fuzzer_test.go) and fixtures under
  [`fuzzing/testdata`](../../fuzzing/testdata). Run `go test -v ./fuzzing/...`.
- Be careful with worker isolation: shared mutable state across workers breaks determinism and parallelism.
- Changes to coverage semantics affect corpus retention and pruning — validate with the corpus scripts in
  [`/scripts`](../../scripts) (see [../operations.md](../operations.md)).
