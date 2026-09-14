# Compilation Configuration

The compilation configuration defines the parameters to use while compiling a target file or project.

### `platform`

- **Type**: String
- **Description**: Refers to the type of platform to be used to compile the underlying target. Currently,
  `crytic-compile` or `solc` can be used as the compilation platform.
- **Default**: `crytic-compile`

### `platformConfig`

- **Type**: Struct
- **Description**: This struct is a platform-dependent structure which offers parameters for compiling the underlying project.
  See below for the structure of `platformConfig` for each compilation platform.
- **Default**: The `platformConfig` for `crytic-compile` is the default value for this struct.

### `platformConfig` for `crytic-compile`

#### `target`

- **Type**: String
- **Description**: Refers to the target that is being compiled.
  > 🚩 Note that if you are using a compilation platform, such as Foundry or Hardhat, the default value for `target`, `.`,
  > should **not** be changed. The `.` is equivalent to telling `crytic-compile` that the entire project needs to compiled,
  > including any dependencies and remappings. In fact, unless you want to compile a single file, that has no third-party
  > imports from, for example, OpenZeppelin, the default value should not be changed.
- **Default**: `.`

#### `solcVersion`

- **Type**: String
- **Description**: Describes the version of `solc` that will be installed and then used for compilation. Note that if you
  are using a compilation platform, such as Foundry or Hardhat, this option does not need to be set.
- **Default**: ""

#### `exportDirectory`

- **Type**: String
- **Description**: Describes the directory where all compilation artifacts should be stored after compilation. Leaving it
  empty will lead to the compilation artifacts being stored in `crytic-export/`.
- **Default**: ""

#### `args`

- **Type**: [String]
- **Description**: Refers to any additional args that one may want to provide to `crytic-compile`. Run `crytic-compile --help`
  to view all of its supported flags. For example, if you would like to specify `--compile-force-framework foundry`, the
  `args` value will be `"args": ["--compile-force-framework", "foundry"]`.
  > 🚩 The `--export-format` and `--export-dir` are already used during compilation with `crytic-compile`.
  > Re-using these flags in `args` will cause the compilation to fail.
- **Default**: `[]`

### `platformConfig` for `solc`

#### `target`

- **Type**: String
- **Description**: Refers to the target that is being compiled. The target must be a single `.sol` file.

## External Library Support

Solidity libraries that are declared as `external` (not inlined by the compiler) require special handling because they must be deployed to specific addresses before contracts that use them can function properly.

### How External Libraries Work

When a contract uses an external library, the compiler inserts a placeholder address in the bytecode. This placeholder must be replaced with the actual deployed address of the library before the contract can be used. Medusa supports two approaches for handling external libraries:

1. **Using `predeployedContracts`** - Deploy libraries to deterministic addresses
2. **Using `--compile-libraries`** - Specify library addresses during compilation

### Method 1: Using `predeployedContracts`

The simplest approach is to use the [`predeployedContracts`](./fuzzing_config.md#predeployedcontracts) configuration option to deploy libraries at specific addresses:

```json
{
  "fuzzing": {
    "targetContracts": ["MyContract"],
    "predeployedContracts": {
      "MathLib": "0x1000",
      "StringUtils": "0x1001"
    }
  }
}
```

When using this method, the libraries are deployed to the specified addresses before any target contracts are deployed, ensuring the library bytecode is available when needed.

### Method 2: Using `--compile-libraries`

For projects with many libraries or complex dependency chains, you can pass library addresses directly to `crytic-compile` using the `args` option:

```json
{
  "compilation": {
    "platform": "crytic-compile",
    "platformConfig": {
      "target": ".",
      "args": [
        "--compile-libraries",
        "(MathLib,0x1000),(StringUtils,0x1001),(ValidationLib,0x1002)"
      ]
    }
  },
  "fuzzing": {
    "predeployedContracts": {
      "MathLib": "0x1000",
      "StringUtils": "0x1001",
      "ValidationLib": "0x1002"
    }
  }
}
```

> 🚩 When using `--compile-libraries`, you must also add the libraries to `predeployedContracts` with the same addresses so that medusa deploys the library bytecode to those addresses before compiling the target contracts.

### Complete Example

Here's a complete example for a project with external libraries:

**Contract with external library:**

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

library MathLib {
    function add(uint256 a, uint256 b) external pure returns (uint256) {
        return a + b;
    }
}

contract Calculator {
    using MathLib for uint256;

    function calculate(uint256 a, uint256 b) public pure returns (uint256) {
        return MathLib.add(a, b);
    }
}
```

**medusa.json configuration:**

```json
{
  "fuzzing": {
    "workers": 10,
    "targetContracts": ["Calculator"],
    "predeployedContracts": {
      "MathLib": "0x1000"
    }
  },
  "compilation": {
    "platform": "crytic-compile",
    "platformConfig": {
      "target": ".",
      "args": ["--compile-libraries", "(MathLib,0x1000)"]
    }
  }
}
```

### Troubleshooting

**Error: Unlinked library reference**

If you see errors about unlinked libraries, ensure that:

1. The library is listed in `predeployedContracts`
2. The library address in `predeployedContracts` matches the address in `--compile-libraries`
3. The library name matches exactly (case-sensitive)

**Error: Library not found at address**

If contracts fail to deploy because libraries are not found:

1. Verify the library is included in `predeployedContracts`
2. Check that the library contract compiles successfully
3. Ensure the deployment order is correct (libraries should be deployed before contracts that use them)
