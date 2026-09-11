# Workflow: The Fuzzing Lifecycle

This page describes what happens end-to-end during a `medusa fuzz` campaign, tying together the CLI, engine, and
chain. It is a condensed, code-anchored version of the user-facing lifecycle doc
[`docs/src/testing/fuzzing_lifecycle.md`](../../docs/src/testing/fuzzing_lifecycle.md); read that for diagrams and
worked examples, and see [../architecture/fuzzing-engine.md](../architecture/fuzzing-engine.md) for component detail.

## 1. Startup & configuration

`main.go` → `cmd.Execute()` → `cmd/fuzz.go` (`cmdRunFuzz`) resolves the `ProjectConfig`:

1. If `--config` is given, load that file (error if unreadable/missing).
2. Else load `medusa.json` from the working directory.
3. Else fall back to the default project config (`fuzzing/config/config_defaults.go`).
4. Finally, explicitly-set CLI flags (`--workers`, `--timeout`, `--deployer`, …) override the loaded config
   (`updateProjectConfigWithFuzzFlags` in [`cmd/fuzz_flags.go`](../../cmd/fuzz_flags.go)).

The config is then handed to `fuzzing.NewFuzzer(config)`. Reference: [`cmd/fuzz.go`](../../cmd/fuzz.go),
[`fuzzing/config/config.go`](../../fuzzing/config/config.go).

## 2. Compilation & target discovery

The fuzzer compiles the target project through the configured platform (`crytic-compile` or `solc`, see
[`/compilation`](../../compilation)), derives `contractDefinitions`, and builds the `baseValueSet` from AST/Slither
constants. Target contracts to fuzz come from `fuzzing.targetContracts` (or all contracts if
`testing.testAllContracts` is true).

## 3. Worker pool

`Fuzzer.Start()` spawns `fuzzing.workers` (default 10) `FuzzerWorker` goroutines, each with an isolated
`chain.TestChain`. See [../architecture/chain-and-state.md](../architecture/chain-and-state.md).

## 4. Deployment (once, on a shared base chain)

Before workers spawn, `Fuzzer.Start()` deploys the target contracts **once** on a base `TestChain` via
`ChainSetupFunc` (using `fuzzing.deployerAddress`), including any contracts dynamically deployed during
construction. Each worker then **clones** that base chain, replaying its deployment blocks, and records the
post-deployment block as `testingBaseBlockIndex` — the **initial deployment state** the worker reverts to between
sequences. Optional pre-deployed / genesis state can be loaded first (see chain docs). Reference: `ChainSetupFunc`
in [`fuzzing/fuzzer_hooks.go`](../../fuzzing/fuzzer_hooks.go).

## 5. The inner fuzzing loop (per worker, repeated)

```
loop until timeout / testLimit / failure (if stopOnFailedTest):
  1. Generate a CallSequence
       - with NewSequenceProbability: a fully random sequence
       - otherwise: mutate corpus entries (head/tail/splice/interleave, with value mutations)
  2. For each element in the sequence:
       a. Execute the call as a transaction (blocks follow each call's fuzzed block/timestamp
          delays — zero-delay calls may share a block)
       b. If coverage increased → add the executed prefix to the corpus
       c. Run test-case hooks (assertion/property/optimization) and update metrics
  3. If a test failed → queue a ShrinkCallSequenceRequest
  4. Revert TestChain to testingBaseBlockIndex (reset state)
```

- A **random transaction** is a call to a random method of a target contract with fuzzed arguments.
- **Coverage** is measured by the coverage tracer; only coverage-increasing sequences are retained.
- Reference: [`fuzzing/fuzzer_worker.go`](../../fuzzing/fuzzer_worker.go),
  [`fuzzing/fuzzer_worker_sequence_generator.go`](../../fuzzing/fuzzer_worker_sequence_generator.go).

## 6. Shrinking a failure

Before generating the next sequence, the worker processes queued shrink requests: it iteratively removes calls and
mutates values while a verifier confirms the test still fails, yielding a **minimal reproduction** (bounded by
`fuzzing.shrinkLimit`). Reference: [`fuzzing/fuzzer_worker_shrinking.go`](../../fuzzing/fuzzer_worker_shrinking.go).

## 7. Worker recycling

After roughly `fuzzing.workerResetLimit` sequences (default 50), a worker is destroyed and recreated to free memory
accumulated in geth's state. Its corpus contributions persist in the shared corpus.

## 8. Corpus pruning (background)

If coverage is enabled and `fuzzing.pruneFrequency` > 0 (default 5 min), a background pruner periodically removes
redundant corpus sequences. Reference: [`fuzzing/corpus/corpus_pruner.go`](../../fuzzing/corpus/corpus_pruner.go)
(the pruner loop; the underlying `PruneSequences` lives in [`corpus.go`](../../fuzzing/corpus/corpus.go)).

## 9. Shutdown & reporting

The campaign ends on timeout, `testLimit`, a failing test (if `testing.stopOnFailedTest`), or an OS interrupt. On
shutdown medusa writes coverage reports (`fuzzing.coverageFormats`), optional revert reports
(`fuzzing.revertReporterEnabled`), and — when coverage is enabled — flushes the corpus to disk. Exit codes are defined in
[`/cmd/exitcodes`](../../cmd/exitcodes). See [../testing.md](../testing.md) for interpreting results.
