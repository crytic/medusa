# Architecture Overview

medusa is a Go program with a Cobra CLI front-end and a fuzzing engine core. This page describes the major
components, how they fit together, and where each lives. For the detailed engine internals see
[fuzzing-engine.md](fuzzing-engine.md); for EVM/state details see [chain-and-state.md](chain-and-state.md).

The canonical prose version of this architecture lives in [`/AGENTS.md`](../../AGENTS.md) ("Architecture & Code
Structure"). This page grounds and links that content to source.

## Components

| Component                | Package                                                                         | Role                                                                                              |
| ------------------------ | ------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| CLI                      | [`/cmd`](../../cmd)                                                             | Parses flags/config, loads `ProjectConfig`, constructs and starts the `Fuzzer`.                   |
| Fuzzer                   | [`/fuzzing`](../../fuzzing) (`fuzzer.go`)                                       | Top-level orchestrator: compilation, worker pool, corpus, coverage, metrics, test case providers. |
| Worker                   | [`/fuzzing`](../../fuzzing) (`fuzzer_worker.go`)                                | One fuzzing thread with an isolated `TestChain`; runs the inner fuzzing loop.                     |
| Sequence generator       | `fuzzing/fuzzer_worker_sequence_generator.go`                                   | Produces new/mutated call sequences using weighted strategies.                                    |
| Corpus                   | [`/fuzzing/corpus`](../../fuzzing/corpus)                                       | Persists coverage-increasing call sequences; prunes redundant entries.                            |
| Coverage                 | [`/fuzzing/coverage`](../../fuzzing/coverage)                                   | EVM tracer + coverage maps + HTML/LCOV report generation.                                         |
| Value generation         | [`/fuzzing/valuegeneration`](../../fuzzing/valuegeneration)                     | Generates/mutates ABI values; maintains the value set.                                            |
| Calls                    | [`/fuzzing/calls`](../../fuzzing/calls)                                         | `CallSequence` / `CallMessage` data structures and execution helpers.                             |
| Test case providers      | `fuzzing/test_case_*`                                                           | Assertion, property, and optimization test detection.                                             |
| Chain                    | [`/chain`](../../chain)                                                         | `TestChain` EVM harness, state factories, cheat codes, tracers.                                   |
| Compilation              | [`/compilation`](../../compilation)                                             | `solc` / `crytic-compile` adapters, ABI utils, artifact hashing.                                  |
| Logging / Events / Utils | [`/logging`](../../logging), [`/events`](../../events), [`/utils`](../../utils) | Cross-cutting infrastructure.                                                                     |

## System flow

```
main.go
  └─ cmd.Execute() → cmd/fuzz.go (cmdRunFuzz)
       └─ load ProjectConfig (medusa.json or defaults)
       ├─ fuzzing.NewFuzzer(config)
       │    ├─ compile targets (crytic-compile / solc)
       │    ├─ derive contract definitions + base value set
       │    └─ register test case providers (assertion/property/optimization)
       └─ Fuzzer.Start()
            └─ spawn N FuzzerWorkers (config.fuzzing.workers)

Per worker (parallel loop):
  1. Create isolated TestChain
  2. Deploy target contracts → record testingBaseBlockIndex
  3. Generate a CallSequence (new or corpus-mutated)
  4. Execute the sequence on the TestChain
  5. Run CallSequenceTestFuncs (test case providers) after each call
  6. On new coverage → add sequence to corpus
  7. On test failure → shrink sequence → report + save
  8. Revert chain to testingBaseBlockIndex → repeat
  9. After WorkerResetLimit sequences → destroy + recreate worker (free memory)
```

Reference: [`fuzzing/fuzzer.go`](../../fuzzing/fuzzer.go), [`fuzzing/fuzzer_worker.go`](../../fuzzing/fuzzer_worker.go),
[`cmd/fuzz.go`](../../cmd/fuzz.go).

## Key architectural patterns

- **Worker-based parallelization.** Each `FuzzerWorker` owns its own `TestChain` with no shared mutable state.
  Workers are periodically destroyed and recreated (`fuzzing.workerResetLimit`, default 50) to bound memory from
  geth's state accumulation. See `FuzzerWorker` in `fuzzing/fuzzer_worker.go`.
- **Event-driven extensibility.** `FuzzerEvents` and `FuzzerWorkerEvents` emit lifecycle events
  (starting/stopping, worker created/destroyed). Test case providers subscribe and register test hooks without
  touching core logic. See [`fuzzing/fuzzer_events.go`](../../fuzzing/fuzzer_events.go),
  [`fuzzing/fuzzer_worker_events.go`](../../fuzzing/fuzzer_worker_events.go), and the generic emitter in
  [`/events`](../../events).
- **Hook-based customization.** `FuzzerHooks` ([`fuzzing/fuzzer_hooks.go`](../../fuzzing/fuzzer_hooks.go)) exposes
  override points: `NewCallSequenceGeneratorConfigFunc`, `NewShrinkingValueMutatorFunc`, `ChainSetupFunc`, and the
  `CallSequenceTestFuncs` list. This is how the Go API extends behavior.
- **Coverage-guided feedback loop.** Only coverage-increasing sequences are retained in the corpus; a background
  pruner keeps it lean. See [fuzzing-engine.md](fuzzing-engine.md).
- **State abstraction.** `MedusaStateDB` / `MedusaStateFactory` abstract the EVM state backend so the same engine
  supports vanilla in-memory state and fork mode. See [chain-and-state.md](chain-and-state.md).

## Where to start when changing things

- Changing **campaign orchestration / lifecycle**: `fuzzing/fuzzer.go`, `fuzzing/fuzzer_worker.go`.
- Changing **how inputs are generated/mutated**: `fuzzing/valuegeneration/`, `fuzzing/fuzzer_worker_sequence_generator.go`.
- Adding a **new kind of test**: add a `test_case_*_provider.go` and register it (see `fuzzing/fuzzer.go`).
- Changing **EVM behavior / cheat codes**: `chain/`.
- Changing **CLI or config**: `cmd/`, `fuzzing/config/`, `chain/config/`.

See [reference/source-map.md](../reference/source-map.md) for a fuller index.
