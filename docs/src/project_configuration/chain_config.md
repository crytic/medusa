# Chain Configuration

The chain configuration defines the parameters for setting up `medusa`'s underlying blockchain.

### `codeSizeCheckDisabled`

- **Type**: Boolean
- **Description**: If `true`, the maximum code size check of 24576 bytes in `go-ethereum` is disabled.
- > 🚩 Setting `codeSizeCheckDisabled` to `false` is not recommended since it complicates the fuzz testing process.
- **Default**: `true`

### `skipAccountChecks`

- **Type**: Boolean
- **Description**: If `true`, account-related checks (nonce validation, transaction origin must be an EOA) are disabled in `go-ethereum`.
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
