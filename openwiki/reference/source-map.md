# Source Map & Change Guide

A navigation index for the medusa codebase, plus "where do I change X" pointers. For narrative explanations, follow
the links into the architecture pages.

## Top-level layout

| Path                                                                                                    | What it is                                                                                                   |
| ------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| [`/main.go`](../../main.go)                                                                             | Process entrypoint → `cmd.Execute()`.                                                                        |
| [`/cmd`](../../cmd)                                                                                     | Cobra CLI: `init`, `fuzz`, `corpus clean`, `completion`; flags; exit codes.                                  |
| [`/fuzzing`](../../fuzzing)                                                                             | Core fuzzing engine (see [engine page](../architecture/fuzzing-engine.md)).                                  |
| [`/chain`](../../chain)                                                                                 | EVM harness `TestChain`, state backends, cheat codes (see [chain page](../architecture/chain-and-state.md)). |
| [`/compilation`](../../compilation)                                                                     | solc / crytic-compile adapters, ABI utils, artifact hashing.                                                 |
| [`/logging`](../../logging)                                                                             | Zerolog-based structured logging.                                                                            |
| [`/events`](../../events)                                                                               | Generic event emitter used for lifecycle hooks.                                                              |
| [`/utils`](../../utils)                                                                                 | Shared helpers (random, reflection, files).                                                                  |
| [`/docs`](../../docs)                                                                                   | mdBook user documentation (`docs/src`).                                                                      |
| [`/scripts`](../../scripts)                                                                             | Corpus diff/stats + docs check (Python).                                                                     |
| [`/AGENTS.md`](../../AGENTS.md), [`/DEV.md`](../../DEV.md), [`/CONTRIBUTING.md`](../../CONTRIBUTING.md) | Contributor guidance.                                                                                        |
| [`/Dockerfile`](../../Dockerfile), [`/flake.nix`](../../flake.nix), [`/.github`](../../.github)         | Build/CI/distribution.                                                                                       |

## `/fuzzing` breakdown

| Path                                                               | Role                                                                    |
| ------------------------------------------------------------------ | ----------------------------------------------------------------------- |
| `fuzzer.go`                                                        | Orchestrator (`Fuzzer`, `NewFuzzer`, `Start`).                          |
| `fuzzer_worker.go`                                                 | Worker + inner fuzzing loop.                                            |
| `fuzzer_worker_sequence_generator.go`                              | Call sequence generation strategies.                                    |
| `fuzzer_worker_shrinking.go`                                       | Failing-sequence shrinking.                                             |
| `fuzzer_hooks.go` / `fuzzer_events.go` / `fuzzer_worker_events.go` | Extensibility hooks & lifecycle events.                                 |
| `fuzzer_metrics.go`                                                | Campaign metrics.                                                       |
| `test_case*.go`, `test_case_*_provider.go`                         | Assertion / property / optimization test types.                         |
| `calls/`                                                           | `CallSequence`, `CallMessage`, execution.                               |
| `corpus/`                                                          | Corpus storage, pruning (`corpus_pruner`), cleaning (`corpus_cleaner`). |
| `coverage/`                                                        | Coverage tracer, maps, source analysis, HTML/LCOV reports.              |
| `valuegeneration/`                                                 | Random/mutational value generators, value set, shrinking mutator.       |
| `contracts/`                                                       | Contract & method definitions derived from compilation.                 |
| `executiontracer/`                                                 | Human-readable execution traces for failures.                           |
| `reverts/`                                                         | Per-function revert metrics reporter.                                   |
| `config/`                                                          | `ProjectConfig`, `FuzzingConfig`, defaults.                             |
| `testdata/`                                                        | Solidity fixtures for end-to-end tests.                                 |

## `/chain` breakdown

| Path                                                                                                                      | Role                                                                                                                 |
| ------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| `test_chain.go`                                                                                                           | Isolated EVM node with snapshots/reverts.                                                                            |
| `test_chain_tracer.go`, `test_chain_deployments_tracer.go`                                                                | `TestChainTracer` interface + deployment/self-destruct detection (the coverage tracer lives in `fuzzing/coverage/`). |
| `test_chain_events.go`                                                                                                    | Chain lifecycle events.                                                                                              |
| `cheat_code_contract.go`, `standard_cheat_code_contract.go`, `console_log_cheat_code_contract.go`, `cheat_code_tracer.go` | Cheat codes (`vm.*`, `console.log`).                                                                                 |
| `config/`                                                                                                                 | `TestChainConfig` (cheat codes, fork, genesis, EVM toggles).                                                         |
| `state/`                                                                                                                  | `MedusaStateFactory` implementations, fork RPC (`state/rpc`), cache (`state/cache`), genesis loader.                 |
| `types/`                                                                                                                  | `MedusaStateDB` interface and chain types.                                                                           |

## `/compilation` breakdown

| Path                                                                         | Role                                                       |
| ---------------------------------------------------------------------------- | ---------------------------------------------------------- |
| `supported_platforms.go`, `compilation_config.go`                            | Platform selection & config.                               |
| `platforms/solc.go`, `platforms/crytic_compile.go`, `platforms/interface.go` | Compilation adapters.                                      |
| `abiutils/`                                                                  | Solidity error/panic decoding and event unpacking helpers. |
| `artifact_hash.go`                                                           | Detects unchanged artifacts to skip recompilation.         |
| `types/`                                                                     | Compilation/artifact/Slither result types.                 |

## "Where do I change X?"

| Goal                                    | Start here                                                                | Tests / checks                                                                                                  |
| --------------------------------------- | ------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| Change campaign orchestration/lifecycle | `fuzzing/fuzzer.go`, `fuzzing/fuzzer_worker.go`                           | `fuzzing/fuzzer_test.go`                                                                                        |
| Change input generation/mutation        | `fuzzing/valuegeneration/`, `fuzzing/fuzzer_worker_sequence_generator.go` | `fuzzing/valuegeneration/*_test.go`, `fuzzing/fuzzer_test.go`                                                   |
| Add a new test case type                | new `fuzzing/test_case_*_provider.go` + register in `fuzzing/fuzzer.go`   | fixtures in `fuzzing/testdata/`, `fuzzing/fuzzer_test.go`                                                       |
| Change coverage / reports               | `fuzzing/coverage/`                                                       | `go test ./fuzzing/...` (no unit tests in `coverage/` itself — exercised via `fuzzer_test.go`), corpus scripts  |
| Add / change a cheat code               | `chain/standard_cheat_code_contract.go`                                   | `TestCheatCodes` in `fuzzing/fuzzer_test.go`, `fuzzing/testdata/contracts/cheat_codes/`, `docs/src/cheatcodes/` |
| Change EVM/state/fork behavior          | `chain/state/`, `chain/config/config.go`                                  | `chain/state/*_test.go`                                                                                         |
| Change CLI or flags                     | `cmd/*.go`                                                                | manual `./medusa --help`, `docs/src/cli/`                                                                       |
| Change config schema/defaults           | `fuzzing/config/config*.go`, `chain/config/`                              | `fuzzing/config/config_test.go`, `docs/src/project_configuration/`, `scripts/check_docs.py`                     |
| Change compilation                      | `compilation/platforms/`                                                  | `compilation/platforms/*_test.go`                                                                               |

Always run `go fmt ./...`, `goimports -w .`, and `golangci-lint run` before committing (see
[../operations.md](../operations.md)). File names should stay ≤ 32 chars per the stated convention (Windows path
limits) — though several existing files exceed it; JSON keys use camelCase.
