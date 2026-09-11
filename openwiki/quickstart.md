# medusa — OpenWiki Quickstart

`medusa` is a cross-platform, [go-ethereum](https://github.com/ethereum/go-ethereum/)-based
**smart contract fuzzer** written in Go, developed by [crytic](https://github.com/crytic) and inspired by
[Echidna](https://github.com/crytic/echidna). It performs **parallelized, coverage-guided fuzz testing** of
Solidity smart contracts, driven either from the CLI or from a Go API for user-extended testing.

Source of truth for this overview: [`/README.md`](../README.md), [`/AGENTS.md`](../AGENTS.md), and
[`/main.go`](../main.go).

> This OpenWiki is an opinionated **map and synthesis layer** over the repository. The project already ships
> extensive user-facing documentation as an mdBook under [`/docs/src`](../docs/src/SUMMARY.md), and a detailed
> agent guide in [`/AGENTS.md`](../AGENTS.md). This wiki tells you _how the codebase is organized and where to
> change things_; the mdBook tells end users _how to use medusa_. Prefer linking to those docs over duplicating them.

## What medusa does

Given one or more Solidity contracts, medusa:

1. Compiles the target project (via `crytic-compile` or `solc`).
2. Spins up a pool of parallel workers, each with an isolated in-process EVM (`TestChain`, built on medusa-geth).
3. Deploys the target contracts and repeatedly generates + executes random/mutated **call sequences**.
4. Runs **test case providers** (assertion, property, optimization) after each call in a sequence to detect invariant violations.
5. Uses **coverage feedback** to keep interesting call sequences in a **corpus** and mutate them to explore deeper.
6. When a test fails, **shrinks** the failing sequence to a minimal reproduction and reports it.

## Key features

- Parallel fuzzing across multiple workers (see `fuzzing.workers`).
- Assertion, property, and optimization testing (built-in test case providers).
- Coverage-guided fuzzing with an on-disk corpus and periodic pruning.
- Mutational value generation seeded from compilation artifacts and runtime values.
- Foundry-style **cheat codes** (`vm.*`) and `console.log` support.
- **Fork mode**: back the EVM state with a remote RPC; separately, a genesis state file can preload Anvil
  `anvil_dumpState` output.
- HTML + LCOV coverage reports and per-function revert reporting.

## Repository layout

| Path                              | Responsibility                                                                                  |
| --------------------------------- | ----------------------------------------------------------------------------------------------- |
| [`/main.go`](../main.go)          | Process entrypoint; delegates to `cmd.Execute()`.                                               |
| [`/cmd`](../cmd)                  | Cobra CLI: `fuzz`, `init`, `corpus clean`, `completion`. Flag parsing and exit codes.           |
| [`/fuzzing`](../fuzzing)          | Core engine: `Fuzzer`, workers, corpus, coverage, value generation, calls, test case providers. |
| [`/chain`](../chain)              | `TestChain` EVM harness, state management/factories, cheat codes, tracers.                      |
| [`/compilation`](../compilation)  | Compilation abstraction with `solc` and `crytic-compile` platform adapters + ABI utils.         |
| [`/logging`](../logging)          | Structured logging (zerolog) with console + file writers.                                       |
| [`/events`](../events)            | Generic event-emitter system used for lifecycle hooks.                                          |
| [`/utils`](../utils)              | Shared helpers (random values, reflection, file ops).                                           |
| [`/docs`](../docs/src/SUMMARY.md) | mdBook user documentation.                                                                      |
| [`/scripts`](../scripts)          | Python helpers for corpus analysis and docs checks.                                             |

## Run it quickly

```bash
go build                       # build the medusa binary
./medusa init                  # scaffold a medusa.json config in the current project
./medusa fuzz                  # run a fuzzing campaign using medusa.json
go run . fuzz --config medusa.json  # compile + run in one step
```

See [operations.md](operations.md) for the full build/test/lint and CLI reference.

## Where to go next

- **[architecture/overview.md](architecture/overview.md)** — components, package map, and system flow.
- **[architecture/fuzzing-engine.md](architecture/fuzzing-engine.md)** — the `fuzzing/` package: fuzzer, workers,
  sequence generation, corpus, coverage, value generation, shrinking, and test case providers.
- **[architecture/chain-and-state.md](architecture/chain-and-state.md)** — the `TestChain`, state factories,
  fork mode, and cheat codes.
- **[workflows/fuzzing-lifecycle.md](workflows/fuzzing-lifecycle.md)** — end-to-end lifecycle of a campaign.
- **[testing.md](testing.md)** — invariant types, writing tests, and running the Go test suite.
- **[operations.md](operations.md)** — building, CLI usage, configuration, and corpus tooling.
- **[reference/source-map.md](reference/source-map.md)** — "I want to change X, where do I look?" index.

## Ground rules for editing this codebase (for agents)

- Follow the conventions in [`/AGENTS.md`](../AGENTS.md): `area: intent` commit format, doc comments on all exported
  symbols, `filepath` (not `path`) for cross-platform correctness, camelCase JSON keys, ≤32-char file names
  (a stated convention — note several existing files exceed it).
- Run `go fmt ./...`, `goimports -w .`, and `golangci-lint run --timeout 5m` before committing.
- Run `go test -v ./...` for verification; target packages with `go test -v ./fuzzing/...`.
- User-facing behavior changes usually require updating [`/docs/src`](../docs/src/SUMMARY.md) — CI runs a docs check
  (`/scripts/check_docs.py`).
