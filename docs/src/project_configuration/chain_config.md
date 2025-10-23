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
