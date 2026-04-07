# Chain Configuration

The chain configuration defines the parameters for setting up `medusa`'s underlying blockchain.

### `codeSizeCheckDisabled`

- **Type**: Boolean
- **Description**: If `true`, the maximum code size check of 24576 bytes in `go-ethereum` is disabled.
  > 🚩 Setting `codeSizeCheckDisabled` to `false` is not recommended since it complicates the fuzz testing process.
- **Default**: `true`

### `skipAccountChecks`

- **Type**: Boolean
- **Description**: If `true`, account-related checks (nonce validation, transaction origin must be an EOA) are disabled in `go-ethereum`.
  > 🚩 Setting `skipAccountChecks` to `false` is not recommended since it complicates the fuzz testing process.
- **Default**: `true`

## Cheatcode Configuration

### `cheatCodesEnabled`

- **Type**: Boolean
- **Description**: Determines whether cheatcodes are enabled.
- **Default**: `true`

### `enableFFI`

- **Type**: Boolean
- **Description**: Determines whether the `ffi` cheatcode is enabled.
  > 🚩 Enabling the `ffi` cheatcode may allow for arbitrary code execution on your machine.
- **Default**: `false`

## Genesis State Configuration

### `genesisStateFile`

- **Type**: String
- **Description**: Path to a JSON file containing EVM state to pre-load before the fuzzing campaign starts.
  This lets you fuzz contracts that were deployed outside of Medusa (e.g., via a Foundry script) without
  replicating their deployment logic. The loaded state — account balances, nonces, bytecode, and storage
  slots — is merged with Medusa's own fuzzer accounts; fuzzer accounts take precedence on address conflicts.

  Three file formats are accepted and detected automatically:

  | Format                  | How to produce it                                                      |
  | ----------------------- | ---------------------------------------------------------------------- |
  | **Native anvil dump**   | `cast rpc anvil_dumpState > state.json` — a gzip-compressed hex string |
  | **Anvil wrapper JSON**  | Decompressed dump with a top-level `"accounts"` field                  |
  | **Plain accounts JSON** | A flat map of `"0xADDR"` → `{balance, nonce, code, storage}`           |

  See [Fuzzing Pre-Deployed Contracts](../testing/genesis_state.md) for a step-by-step workflow.

- **Default**: `""`

### `genesisContractMappings`

- **Type**: `{"contractAddress": "contractName"}` (e.g. `{"0x5FbDB2315678afecb367f032d93F642f64180aa3": "Counter"}`)
- **Description**: Maps addresses in the genesis state to contract names from the compilation artifacts.
  Without this mapping, Medusa loads the bytecode but has no ABI for it and cannot generate calls.

  For each entry, Medusa will:
  1. Skip deploying the named contract (it is already present in the genesis state).
  2. Bind the compiled ABI to the given address.

  **This controls ABI binding only, not which contracts are fuzzed.** Listing a contract here makes
  its functions callable (by the fuzzer or by other contracts), but does not make the fuzzer generate
  calls to it directly. To fuzz a genesis contract, also add its name to `targetContracts`.

  Contracts that are dependencies of your targets (tokens, oracles, registries, etc.) should appear
  here but not in `targetContracts` — they need an ABI so internal calls resolve correctly, but you
  do not want the fuzzer flooding them with top-level calls.

- **Default**: `{}`

## Fork Configuration

### `forkModeEnabled`

- **Type**: Boolean
- **Description**: Determines whether fork mode is enabled
- **Default**: `false`

### `rpcUrl`

- **Type**: String
- **Description**: Determines the RPC URL that will be queried during fork mode.
- **Default**: ""

### `rpcBlock`

- **Type**: Integer
- **Description**: Determines the block height that fork state will be queried for. Block tags like `LATEST` are not supported yet.
- **Default**: `1`

### `poolSize`

- **Type**: Integer
- **Description**: Determines the size of the client pool used to query the RPC. It is recommended to use a pool size
- that is 2-3x the number of workers used, but smaller pools may be required to avoid exceeding external RPC query limits.
- **Default**: `20`
