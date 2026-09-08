# Operations: Build, CLI, Configuration & Tooling

This page is the practical reference for building medusa, running it, configuring it, and using its developer
tooling. It complements [`/AGENTS.md`](../AGENTS.md) and [`/DEV.md`](../DEV.md).

## Build

```bash
go build                 # compile the medusa CLI binary
./medusa --help          # verify the built binary
nix build                # build with Nix (update flake.nix vendorHash after go.mod changes)
nix develop              # enter the pinned Nix dev shell with all deps
```

The version string lives in [`cmd/root.go`](../cmd/root.go) (and is bumped in `flake.nix` on release). Entrypoint:
[`main.go`](../main.go) → `cmd.Execute()`.

## Lint & format (run before committing)

```bash
goimports -w .                       # order imports (stdlib, then external; no -local grouping configured)
go fmt ./...                         # format Go
golangci-lint run --timeout 5m       # comprehensive lint (mirrors CI)
dprint fmt                           # format markdown/YAML/JSON (config: dprint.json)
actionlint                           # lint GitHub Actions workflows
```

Pre-commit hooks automate these (`prek install` once, then `prek run`). Config:
[`.pre-commit-config.yaml`](../.pre-commit-config.yaml). Docs consistency is enforced in CI by
[`scripts/check_docs.py`](../scripts/check_docs.py).

## CLI commands (`/cmd`)

Built with Cobra. Root command in [`cmd/root.go`](../cmd/root.go).

| Command                  | File                                        | Purpose                                                              |
| ------------------------ | ------------------------------------------- | -------------------------------------------------------------------- |
| `medusa init [platform]` | [`cmd/init.go`](../cmd/init.go)             | Scaffold a `medusa.json` project config (optionally for a platform). |
| `medusa fuzz`            | [`cmd/fuzz.go`](../cmd/fuzz.go)             | Run a fuzzing campaign (uses `medusa.json` or `--config`).           |
| `medusa corpus clean`    | [`cmd/corpus.go`](../cmd/corpus.go)         | Remove invalid/unreplayable sequences from a corpus.                 |
| `medusa completion`      | [`cmd/completion.go`](../cmd/completion.go) | Generate shell completion scripts.                                   |

Common examples:

```bash
medusa init                      # create default medusa.json
medusa fuzz                      # run using medusa.json in cwd
medusa fuzz --config custom.json # run with an explicit config
go run . --config medusa.json    # compile + run from source
```

Many `fuzz` flags override config fields (see [`cmd/fuzz_flags.go`](../cmd/fuzz_flags.go)). Exit codes are defined in
[`/cmd/exitcodes`](../cmd/exitcodes). Full CLI docs: [`docs/src/cli/overview.md`](../docs/src/cli/overview.md).

## Configuration

The top-level config is `ProjectConfig` ([`fuzzing/config/config.go`](../fuzzing/config/config.go)), serialized as
`medusa.json` with four top-level sections (`fuzzing`, `compilation`, `slither`, `logging`); `testing` and
`chainConfig` are sub-objects nested **inside** `fuzzing`:

| Section               | Struct                          | Highlights (with defaults from `config_defaults.go`)                                                                                                                                                                                                                                                  |
| --------------------- | ------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `fuzzing`             | `FuzzingConfig`                 | `workers` (10), `workerResetLimit` (50), `callSequenceLength` (100), `shrinkLimit` (5000), `timeout` (0=∞), `testLimit` (0=∞), `pruneFrequency` (5 min), `coverageEnabled` (true), `coverageFormats` (`html`,`lcov`), `targetContracts`, `deployerAddress`, `senderAddresses`, `transactionGasLimit`. |
| `fuzzing.testing`     | `TestingConfig`                 | `stopOnFailedTest`, `testViewMethods`, `verbosity`, `assertionTesting`, `propertyTesting` (prefix `property_`), `optimizationTesting` (prefix `optimize_`), target/exclude function signatures.                                                                                                       |
| `fuzzing.chainConfig` | `chain/config.TestChainConfig`  | `cheatCodes`, `forkConfig` (fork mode), `genesisStateFile`/`genesisContractMappings`, `contractAddressOverrides`, `codeSizeCheckDisabled`, `skipAccountChecks`.                                                                                                                                       |
| `compilation`         | `compilation.CompilationConfig` | Platform (`crytic-compile`/`solc`) and platform-specific args.                                                                                                                                                                                                                                        |
| `slither`             | `types.SlitherConfig`           | Slither integration used to seed constants into the value set.                                                                                                                                                                                                                                        |
| `logging`             | `LoggingConfig`                 | Single `level` for console/file output, `logDirectory` (non-empty enables file logging), `noColor`.                                                                                                                                                                                                   |

Per-field reference (user docs): [`docs/src/project_configuration/overview.md`](../docs/src/project_configuration/overview.md)
and its sub-pages. See [architecture/chain-and-state.md](architecture/chain-and-state.md) for fork/genesis details.

## Corpus tooling (`/scripts`)

> Per [`/AGENTS.md`](../AGENTS.md): only run these when explicitly needed; most changes don't touch them.

```bash
python3 scripts/corpus_diff.py corpus1 corpus2   # methods present in one corpus but not the other
python3 scripts/corpus_stats.py corpus           # sequence count, avg length, method frequency
```

See [`/DEV.md`](../DEV.md) for example output.

## Deployment / distribution

- Install: `go install github.com/crytic/medusa@latest`.
- Container: [`Dockerfile`](../Dockerfile).
- Nix flake: [`flake.nix`](../flake.nix) / [`flake.lock`](../flake.lock). Update `vendorHash` after `go.mod`
  changes (`nix build` reports the correct hash).
- CI workflows live under [`.github/workflows`](../.github/workflows).
