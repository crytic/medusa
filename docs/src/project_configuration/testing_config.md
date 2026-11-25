# Testing Configuration

The testing configuration can be broken down into a few subcomponents:

- **High-level configuration**: Configures global testing parameters, regardless of the type of testing.
- **Assertion testing configuration**: Configures what kind of EVM panics should be treated as a failing fuzz test.
- **Property testing configuration**: Configures what kind of function signatures should be treated as property tests.
- **Optimization testing configuration**: Configures what kind of function signatures should be treated as optimization tests.

We will go over each subcomponent one-by-one:

## High-level Configuration

### `stopOnFailedTest`

- **Type**: Boolean
- **Description**: Determines whether the fuzzer should stop execution after the first _failed_ test. If `false`, `medusa`
  will continue fuzzing until either the [`testLimit`](./fuzzing_config.md#testlimit) is hit, the [`timeout`](./fuzzing_config.md#timeout)
  is hit, or the user manually stops execution.
- **Default**: `true`

### `stopOnFailedContractMatching`

- **Type**: Boolean
- **Description**: Determines whether the fuzzer should stop execution if it is unable to match the bytecode of a dynamically
  deployed contract. A dynamically deployed contract is one that is created during the fuzzing campaign
  (versus one that is specified in the [`fuzzing.targetContracts`](./fuzzing_config.md#targetcontracts)).
  Here is an example of a dynamically deployed contract:

```solidity

contract MyContract {
  OtherContract otherContract;
  constructor() {
    // This is a dynamically deployed contract
    otherContract = new otherContract();
  }
}
```

- **Default**: `false`

### `stopOnNoTests`

- **Type**: Boolean
- **Description**: Determines whether the fuzzer should stop execution if no tests are found
  (property tests, assertion tests, optimization tests, or custom API-level tests). If `false` and no tests are found,
  `medusa` will continue fuzzing until either the [`testLimit`](./fuzzing_config.md#testlimit) is hit,
  the [`timeout`](./fuzzing_config.md#timeout) is hit, or the user manually stops execution.
- **Default**: `true`

### `testViewMethods`

- **Type**: Boolean
- **Description**: Whether `pure` / `view` functions should be called and tested.
- **Default**: `true`

### `testAllContracts`

- **Type**: Boolean
- **Description**: Determines whether all contracts should be tested (including dynamically deployed ones), rather than
  just the contracts specified in the project configuration's [`fuzzing.targetContracts`](./fuzzing_config.md#targetcontracts).
- **Default**: `false`

### `verbosity`:

- **Type**: Enum (VerbosityLevel)
- **Description**: Controls the level of detail in execution traces:
  - **Verbose (0)**: Only shows top-level transactions in the execution trace. Only events in the top-level call frame and return data are included.
  - **VeryVerbose (1)**: Shows nested calls with standard detail (default behavior).
  - **VeryVeryVerbose (2)**: Shows all call sequence elements with maximum detail, attaching traces to every call in the sequence.
- **CLI flags**: Use `-v`, `-vv`, or `-vvv` to set verbosity levels 0, 1, or 2 respectively.
- **Default**: `VeryVerbose (1)`

### `targetFunctionSignatures`:

- **Type**: [String]
- **Description**: A list of function signatures that the fuzzer should exclusively target by omitting calls to other signatures. The signatures should specify the contract name and signature in the ABI format like `Contract.func(uint256,bytes32)`.
  > **Note**: Property and optimization tests will always be called even if they are not explicitly specified in this list.
- **Default**: `[]`

### `excludeFunctionSignatures`:

- **Type**: [String]
- **Description**: A list of function signatures that the fuzzer should exclude from the fuzzing campaign. The signatures should specify the contract name and signature in the ABI format like `Contract.func(uint256,bytes32)`.
  > **Note**: Property and optimization tests will always be called and cannot be excluded.
- **Default**: `[]`

## Assertion Testing Configuration

### `enabled`

- **Type**: Boolean
- **Description**: Enable or disable assertion testing
- **Default**: `true`

### `panicCodeConfig`

- **Type**: Struct
- **Description**: This struct describes the various types of EVM-level panics that should be considered a "failing case".
  By default, only an `assert(false)` is considered a failing case. However, these configuration options would allow a user
  to treat arithmetic overflows or division by zero as failing cases as well.

#### `failOnAssertion`

- **Type**: Boolean
- **Description**: Triggering an assertion failure (e.g. `assert(false)`) should be treated as a failing case.
- **Default**: `true`

#### `failOnCompilerInsertedPanic`

- **Type**: Boolean
- **Description**: Triggering a compiler-inserted panic should be treated as a failing case.
- **Default**: `false`

#### `failOnArithmeticUnderflow`

- **Type**: Boolean
- **Description**: Arithmetic underflow or overflow should be treated as a failing case
- **Default**: `false`

#### `failOnDivideByZero`

- **Type**: Boolean
- **Description**: Dividing by zero should be treated as a failing case
- **Default**: `false`

#### `failOnEnumTypeConversionOutOfBounds`

- **Type**: Boolean
- **Description**: An out-of-bounds enum access should be treated as a failing case
- **Default**: `false`

#### `failOnIncorrectStorageAccess`

- **Type**: Boolean
- **Description**: An out-of-bounds storage access should be treated as a failing case
- **Default**: `false`

#### `failOnPopEmptyArray`

- **Type**: Boolean
- **Description**: A `pop()` operation on an empty array should be treated as a failing case
- **Default**: `false`

#### `failOnOutOfBoundsArrayAccess`

- **Type**: Boolean
- **Description**: An out-of-bounds array access should be treated as a failing case
- **Default**: `false`

#### `failOnAllocateTooMuchMemory`

- **Type**: Boolean
- **Description**: Overallocation/excessive memory usage should be treated as a failing case
- **Default**: `false`

#### `failOnCallUninitializedVariable`

- **Type**: Boolean
- **Description**: Calling an uninitialized variable should be treated as a failing case
- **Default**: `false`

### `failOnCustomErrors`

- **Type**: [String]
- **Description**: A list of Solidity [custom errors](https://docs.soliditylang.org/en/latest/contracts.html#errors-and-the-revert-statement)
  that should be treated as a failing case. If a tested method reverts with one of these errors, an assertion failure is
  reported (and the call sequence is shrunk and saved as a reproducer), just as it would be for an `assert(false)`.
  Each entry may either be an error signature (e.g. `InsufficientBalance(uint256,uint256)`) or a 4-byte error selector
  in hex (e.g. `0xcf479181`). Signatures must not contain parameter names or whitespace. Type aliases such as `uint`,
  `int` and `byte` are canonicalized (including within arrays and tuples), so `InsufficientBalance(uint,uint)` is
  equivalent to the example above. User-defined types must be written as their ABI types: a tuple for a struct
  (e.g. `(address,uint256)`), `uint8` for an enum, and `address` for a contract or interface. Invalid signatures are
  rejected when the configuration is loaded.
  > **Note**: Only the revert data returned by the tested method is checked. A custom error raised in a nested call is
  > only detected if it is bubbled up (re-thrown) by the tested method.
- **Default**: `[]`

## Property Testing Configuration

### `enabled`

- **Type**: Boolean
- **Description**: Enable or disable property testing.
- **Default**: `true`

### `testPrefixes`

- **Type**: [String]
- **Description**: The list of prefixes that the fuzzer will use to determine whether a given function is a property test or not.
  For example, if `property_` is a test prefix, then any function name in the form `property_*` may be a property test.
  > **Note**: If you are moving over from Echidna, you can add `echidna_` as a test prefix to quickly port over the property tests from it.
- **Default**: `[property_]`

## Optimization Testing Configuration

### `enabled`

- **Type**: Boolean
- **Description**: Enable or disable optimization testing.
- **Default**: `true`

### `testPrefixes`

- **Type**: [String]
- **Description**: The list of prefixes that the fuzzer will use to determine whether a given function is an optimization
  test or not. For example, if `optimize_` is a test prefix, then any function name in the form `optimize_*` may be a property test.
- **Default**: `[optimize_]`
