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

### `testAllContracts`

- **Type**: Boolean
- **Description**: Determines whether all contracts should be tested (including dynamically deployed ones), rather than
  just the contracts specified in the project configuration's [`fuzzing.targetContracts`](./fuzzing_config.md#targetcontracts).
- **Default**: `false`

### `traceAll`:

- **Type**: Boolean
- **Description**: Determines whether an [execution trace](TODO) should be attached to each element of a call sequence
  that triggered a test failure.
- **Default**: `false`

## Assertion Testing Configuration

### `enabled`

- **Type**: Boolean
- **Description**: Enable or disable assertion testing
- **Default**: `true`

### `testViewMethods`

- **Type**: Boolean
- **Description**: Whether `pure` / `view` functions should be tested for assertion failures.
  > 🚩 Fuzzing `pure` and `view` functions is not currently implemented. Thus, enabling this option to `true` does not
  > update the fuzzer's behavior.
- **Default**: `false`

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
