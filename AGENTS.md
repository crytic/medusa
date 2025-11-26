# AGENTS.md

This file provides guidance to LLM Agents (e.g. Claude Code or Codex) when working with code in this repository.

## Project Overview

Medusa is a cross-platform smart contract fuzzer written in Go, built on go-ethereum and inspired by Echidna. It provides parallelized, coverage-guided fuzz testing of Solidity smart contracts through CLI or Go API. The fuzzer supports assertion testing, property testing, optimization testing, and cheat codes.

## Development Commands

### Build

- `go build` - Compile the CLI binary for local testing
- `nix build` - Build using Nix (run after modifying `go.mod` to refresh vendorHash)
- `./medusa --help` - Verify the built binary

### Test

- `go test -v ./...` - Run all unit and integration tests
- `go test -v ./fuzzing/...` - Run tests for a specific package with verbose output
- `go test -v -run TestName ./package/...` - Run a specific unit test by name
- `go test -cover ./...` - Run tests with coverage metrics

### Lint & Format

- `go fmt ./...` - Format Go code (required before commits)
- `golangci-lint run --timeout 5m` - Run comprehensive lint checks (mirrors CI)
- `prettier '**/*.md' '**/*.yml' '**/*.json' -w` - Format markdown, YAML, and JSON files
- `actionlint` - Lint GitHub Actions workflow files

### Run

- `go run . --config path/to/config.yaml` - Compile and run the fuzzer
- `nix develop` - Enter the pinned Nix development shell with all dependencies

## Architecture & Code Structure

### Module Responsibilities

- **`cmd/`** - CLI entry point using Cobra framework. Defines `fuzz`, `init`, and `completion` commands.
- **`fuzzing/`** - Core fuzzing orchestration including workers, corpus management, coverage tracking, value generation, and test case providers.
- **`chain/`** - EVM test harness (TestChain) built on medusa-geth with state management and cheat code support (vm.\* functions).
- **`compilation/`** - Smart contract compilation abstraction with platform adapters for solc and crytic-compile.
- **`logging/`** - Structured logging primitives using Zerolog.
- **`events/`** - Event emitter system for fuzzer lifecycle hooks and extensibility.
- **`utils/`** - Shared utilities for random values, reflection, file operations.

### High-Level System Flow

```
CLI (main.go)
  ↓
cmd/fuzz.go (Cobra Command)
  ↓
fuzzing.NewFuzzer(ProjectConfig)
  ├─→ Compilation (compile contracts via crytic-compile/solc)
  ├─→ FuzzerWorker pool (N parallel workers)
  ├─→ TestChain per worker (isolated EVM instance)
  ├─→ Corpus (loads/saves coverage-increasing call sequences)
  ├─→ Test Case Providers (Assertion, Property, Optimization)
  └─→ Coverage Tracer (if enabled)

Worker Loop (per worker, in parallel):
  1. Deploy contracts via ChainSetupFunc hook
  2. Generate CallSequence (mutations from corpus)
  3. Execute sequence on TestChain
  4. Run CallSequenceTestFuncs (test case providers)
  5. If test fails → shrink sequence → save to corpus
  6. Update coverage maps
  7. Reset chain state to base block → repeat
```

### Key Architectural Patterns

**Event-Driven Extensibility**: `FuzzerEvents` emits lifecycle events (FuzzerStarting, FuzzerStopping, WorkerCreated, WorkerDestroyed). Test case providers subscribe to these events and register `CallSequenceTestFunc` hooks, enabling new test types without modifying core logic.

**Hook-Based Customization**: `FuzzerHooks` provides customization points:

- `NewCallSequenceGeneratorConfigFunc` - Customize sequence generation strategy
- `NewShrinkingValueMutatorFunc` - Customize shrinking heuristics
- `ChainSetupFunc` - Customize deployment and initialization logic
- `CallSequenceTestFuncs[]` - Register test functions to run after each sequence

**Coverage-Guided Fuzzing**: The corpus stores only call sequences that increase coverage. Sequences are mutated using weighted strategies (corpus head/tail, splice, interleave, new). A corpus pruner periodically removes redundant sequences to keep the corpus lean.

**Worker-Based Parallelization**: Each `FuzzerWorker` has its own isolated `TestChain` instance with no shared state. Workers are periodically destroyed and recreated (`WorkerResetLimit`) to prevent memory bloat from geth's state accumulation.

**Call Sequence Shrinking**: When a test fails, a `ShrinkCallSequenceRequest` is generated. The worker iteratively removes/mutates calls while a `VerifierFunction` confirms the test still fails, producing a minimal reproduction.

**State Management**: `TestChain` rebases to `testingBaseBlockIndex` after each sequence execution, providing clean state for the next sequence. `MedusaStateDB` interface abstracts underlying state implementation (supports native geth StateDB and fork mode).

## Key Abstractions

### Core Data Structures

- **`CallSequence`** - Array of `CallSequenceElement` representing a transaction sequence to execute on the test chain.
- **`CallSequenceElement`** - Single call with target address, method, arguments, block delay, and gas limit.
- **`TestCase` interface** - Represents a test with Status (NOT_STARTED, RUNNING, PASSED, FAILED), Name, CallSequence, and result Message.
- **`Corpus`** - Persistent storage of coverage-increasing call sequences with pruning. Stored on disk in `corpus/` directory.
- **`CoverageMaps`** - Tracks branch coverage per contract (including jumps, returns, reverts, and contract entrance), used to identify coverage-increasing sequences.
- **`FuzzerWorker`** - Single execution thread with its own TestChain, deployed contracts, and sequence generator.
- **`ProjectConfig`** - Top-level configuration containing FuzzingConfig, CompilationConfig, LoggingConfig.
- **`FuzzingConfig`** - Workers count, timeout, test limit, corpus directory, coverage settings, test case configurations.

### Test Case Providers

Three built-in test case providers run concurrently:

1. **AssertionTestCaseProvider**: Monitors EVM-level panic conditions for each contract method (e.g., `assert()`, arithmetic underflow/overflow, divide by zero, array access violations). Which panic codes trigger test failures is configured via `PanicCodeConfig` in `FuzzingConfig.Testing.AssertionTesting`.
2. **PropertyTestCaseProvider**: Calls property test functions (prefix-based naming, must be view functions returning bool). If a property returns false, the test fails.
3. **OptimizationTestCaseProvider**: Tracks optimization targets (e.g., maximize gas usage) and shrinks sequences to find minimal paths to targets.

## Testing Guidelines

### Test Organization

- Co-locate tests beside implementations using `*_test.go` suffix
- Use table-driven tests for deterministic logic
- Add `t.Parallel()` for parallelizable tests
- Document invariants with assertions

### Running Tests

- `go test -v ./...` - Run all tests
- `go test -v ./fuzzing/...` - Target specific packages
- `go test -v -run TestName ./package/...` - Run a specific unit test

### Corpus Analysis

Explicitly ask the user if these scripts should be run. In more cases than not, these scripts don't need to be touched.

- Use `python3 scripts/corpus_diff.py old new` to compare corpora and identify method coverage differences
- Use `python3 scripts/corpus_stats.py corpus` to generate statistics (sequence count, average length, method frequency)

## Contribution Workflow

### Branch & Commit Conventions

- Branch naming: `dev/<short-scope>` (e.g., `dev/coverage-reports`)
- Commit format: `area: intent` (e.g., `fuzzing: tighten revert handling`)
- Reference issues with `(#123)` when applicable

### Code Standards

- **Naming**: Packages lowercase (`fuzzing`, `chain`), exported types UpperCamelCase (`TestChain`), private helpers lowerCamel (`newWorker`)
- **Documentation**: All exported symbols need doc comments. Add inline comments for complex logic.
- **File naming**: Max 32 characters to avoid Windows path length issues
- **JSON keys**: Use camelCase rather than snake_case

### Cross-Platform Considerations

- Code must work on Linux, macOS, and Windows
- Use `filepath` package (not `path`) for file operations - respects system path separators
- Support both LF (`\n`) and CRLF (`\r\n`) line endings when processing text files
- Test on multiple platforms before submitting PRs

### Nix Workflow

- When dependencies change, update `vendorHash` in `flake.nix`
- Run `nix build` - if vendorHash needs updating, Nix will report the correct hash to use
- Replace the `specified` value with the `got` value from the error message

### Pre-Commit Checklist

1. `go fmt ./...` - Format Go code
2. `golangci-lint run --timeout 5m` - Run linter
3. `go test -v ./...` - Run all tests
4. `prettier '**/*.md' '**/*.yml' '**/*.json' -w` - Format markdown/JSON/YAML
5. Verify changes work on target platforms
