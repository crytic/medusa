# Chain & State (`/chain`)

The [`/chain`](../../chain) package is medusa's in-process EVM harness. Each `FuzzerWorker` owns one `TestChain`
instance, giving every worker an isolated blockchain to deploy contracts and execute call sequences against. It is
built on **medusa-geth** (crytic's fork of go-ethereum).

## TestChain (`test_chain.go`)

`TestChain` is the isolated EVM node abstraction. Key responsibilities:

- Maintain a chain of blocks and account state; produce blocks for each call in a sequence.
- Support **snapshots / reverts** so the worker can rebase to `testingBaseBlockIndex` after each sequence (cheap
  state reset instead of re-deploying).
- Install tracers: the deployments tracer (`test_chain_deployments_tracer.go`) detects dynamically deployed
  contracts, and the coverage tracer records coverage.
- Emit chain events (`test_chain_events.go`) consumed by the fuzzer.

Reference: [`chain/test_chain.go`](../../chain/test_chain.go),
[`chain/test_chain_tracer.go`](../../chain/test_chain_tracer.go).

## State backends (`/chain/state`)

State is abstracted behind two interfaces so the engine is agnostic to where account state comes from:

- **`MedusaStateDB`** (in [`/chain/types`](../../chain/types)) — the EVM state database interface.
- **`MedusaStateFactory`** (`state/factories.go`) — thread-safe factory for creating state DBs, allowing globally
  shared data (like RPC caches) across all `TestChain` instances.

Three factory implementations exist ([`chain/state/factories.go`](../../chain/state/factories.go)):

| Factory                 | Backend                             | Used for                                                |
| ----------------------- | ----------------------------------- | ------------------------------------------------------- |
| `VanillaStateDbFactory` | Native geth `state.New`             | Normal in-memory fuzzing.                               |
| `ForkedStateFactory`    | Remote RPC via `stateBackend`       | **Fork mode** — lazily fetch state from a live network. |
| `UnbackedStateFactory`  | `EmptyBackend` with forked-DB logic | Forked-DB semantics without a real remote source.       |

### Fork mode (`/chain/state/rpc`, `/chain/state/cache`)

When `chainConfig.forkConfig.forkModeEnabled` is set, medusa backs EVM state with a remote RPC:

- `remote_state_provider.go` lazily fetches accounts/storage/code from the RPC as the EVM touches them.
- `rpc/client_pool.go` manages a pool of RPC clients (`forkConfig.poolSize`); recent history includes memory-leak
  fixes in the fork-mode RPC pool.
- `cache/persistent_cache.go` caches remote state on disk so repeated runs avoid redundant RPC calls.

Fork settings: `forkConfig.rpcUrl`, `forkConfig.rpcBlock`, `forkConfig.poolSize`. Defined in
[`chain/config/config.go`](../../chain/config/config.go); consumed by
[`chain/state/rpc_backend.go`](../../chain/state/rpc_backend.go).

### Genesis / pre-deployed state (`genesis_loader.go`)

`chainConfig.genesisStateFile` loads a genesis allocation so you can fuzz contracts deployed **outside** medusa
(e.g. via a Foundry/Anvil deployment). Formats auto-detected: native `anvil_dumpState`, decompressed anvil wrapper
JSON, and plain accounts JSON. `chainConfig.genesisContractMappings` maps genesis addresses to compiled contract
names so the fuzzer knows which ABIs/functions to call. Fuzzer-generated accounts take precedence over loaded state.
Reference: [`chain/state/genesis_loader.go`](../../chain/state/genesis_loader.go) and the user doc
[`docs/src/testing/genesis_state.md`](../../docs/src/testing/genesis_state.md).

## Cheat codes (`cheat_code_*`, `standard_cheat_code_contract.go`)

Cheat codes are Foundry-style `vm.*` functions implemented as **pre-compiled contracts**:

- `CheatCodeContract` (`cheat_code_contract.go`) registers a table of function selectors → handlers.
- `standard_cheat_code_contract.go` implements the standard `vm.*` cheat codes (`warp`, `roll`, `prank`, `deal`,
  `etch`, `store`/`load`, `sign`, parsing helpers, `ffi`, `assert*`, snapshots, etc.).
- `console_log_cheat_code_contract.go` implements `console.log` support.
- `cheat_code_tracer.go` provides the EVM execution hooks cheat codes rely on (e.g. prank scoping).

Cheat codes are gated by `chainConfig.cheatCodes` (`cheatCodesEnabled`, and `enableFFI` for the dangerous `ffi`
code). The authoritative per-cheatcode reference is the mdBook:
[`docs/src/cheatcodes/cheatcodes_overview.md`](../../docs/src/cheatcodes/cheatcodes_overview.md).

## Change-oriented notes

- Adding a cheat code: implement a handler in `standard_cheat_code_contract.go` (or a new contract), register the
  selector, and document it under `docs/src/cheatcodes/`. Tests live in
  [`chain/test_chain_test.go`](../../chain/test_chain_test.go) and `fuzzing/testdata/contracts/cheat_codes/`.
- Fork-mode changes: watch for memory leaks and cache correctness; tests under `chain/state/` cover providers,
  factories, and the genesis loader.
- EVM behavior toggles live in [`chain/config/config.go`](../../chain/config/config.go)
  (`codeSizeCheckDisabled`, `skipAccountChecks`, `contractAddressOverrides`).
