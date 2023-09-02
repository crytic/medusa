# Project Configuration

The project configuration will dictate the compilation, fuzzing, and logging parameters for `medusa`. The fuzzing parameters are determined by the **fuzzing configuration**, the compilation parameters are determined by the **compilation configuration**, and the logging parameters are determined by the **logging configuration**. Thus, a _project_ configuration is a combination of all three configurations.

Run `medusa init` using [`medusa`'s CLI](https://github.com/crytic/medusa/wiki/Command-Line-Interface) to generate a `medusa.json` with the default value for each configuration. See [Example Project Configuration File](https://github.com/crytic/medusa/wiki/Example-Project-Configuration-File) for an example `medusa.json`.

Now, we will walk through each configuration (fuzzing, compilation, and logging) and review the various options for each one.

## Fuzzing Configuration

The fuzzing configuration defines the parameters for the fuzzing campaign:

- `workers`
  - **Type**: Integer
  - **Description**: The number of worker threads to parallelize fuzzing operations on.
  - **Default**: 10
- `workerResetLimit`
  - **Type**: Integer
  - **Description**: The number of call sequences a worker should process on its underlying chain before being fully reset, freeing memory. After resetting, the worker will be re-created and continue processing of call sequences.
    > 🚩 This setting, along with `workers` influence the speed and memory consumption of the fuzzer. Setting this value higher will result in greater memory consumption per worker. Setting it too high will result in the in-memory chain's database growing to a size that is slower to process. Setting it too low may result in frequent worker resets that are computationally expensive for complex contract deployments that need to be replayed during worker reconstruction.
  - **Default**: 50
- `timeout`
  - **Type**: Integer
  - **Description**: The number of seconds before the fuzzing campaign should be terminated. If a zero value is provided, the timeout will not be enforced. The timeout begins counting after compilation succeeds and the fuzzing campaign has started.
  - **Default**: 0
- `testLimit`
  - **Type**: Unsigned Integer
  - **Description**: The number of function calls to make before the fuzzing campaign should be terminated. If a zero value is provided, no call limit will be enforced.
  - **Default**: 0
- `callSequenceLength`
  - **Type**: Integer
  - **Description**: The maximum number of function calls to generate in a single call sequence in the attempt to violate properties. After every `callSequenceLength` function calls, the blockchain is reset for the next sequence of transactions.
  - **Default**: 100
- `coverageEnabled`
  - **Type**: Boolean
  - **Description**: Whether coverage-increasing call sequences should be saved in the corpus for the fuzzer to mutate / re-use. This aims to help achieve greater coverage across contracts when testing.
  - **Default**: `true`
- `corpusDirectory`
  - **Type**: String
  - **Description**: The file path where the corpus should be saved. The corpus collects artifacts during a fuzzing campaign that help drive fuzzer features (e.g. a call sequence that increases code coverage is stored in the corpus). This sequence can then be re-used / mutated by the fuzzer during the next fuzzing campaign.
  - **Default**: ""
- `deploymentOrder`
  - **Type**: [String] (e.g. `["FirstContract", "SecondContract", "ThirdContract"]`)
  - **Description**: A list of contract names that indicates the order in which the compiled contracts (contracts resulting from compilation) should be deployed to the fuzzer on startup. **If there is more than one contract in the target system, at least one contract name must be specified here.**
    > 🚩 Changing this order may render entries in the corpus invalid. It is recommended to clear your corpus when doing so.
  - **Default**: []
- `constructorArgs`
  - **Type**: Mapping of contract to a mapping of variable name to value (contract => variable => value)
  - **Description**: If a contract in the `deploymentOrder` has a constructor that takes in variables, these can be specified here. An example can be found [here](TODO).
  - **Default**: Empty mapping
- `deployerAddress`
  - **Type**: Address
  - **Description**: The address used to deploy contracts on startup, represented as a hex string.
    > 🚩 Changing this address may render entries in the corpus invalid. It is recommended to clear your corpus when doing so.
  - **Default**: `0x30000`
- `senderAddresses`
  - **Type**: [Address]
  - **Description**: Defines the account addresses used to send function calls to deployed contracts in the fuzzing campaign.
    > 🚩 Changing these addresses may render entries in the corpus invalid. It is recommended to clear your corpus when doing so.
  - **Default**: `[0x10000, 0x20000, 0x30000]`
- `blockNumberDelayMax`
  - **Type**: Unsigned Integer
  - **Description**: Defines the maximum block number jump the fuzzer should make between test transactions. The fuzzer will use this value to make the next block's block number between `[1, blockNumberDelayMax]` more than that of the previous block. Jumping block numbers allows `medusa` to enter code paths that require a given number of blocks to pass.
  - **Default**: 60_480
- `blockTimestampDelayMax`
  - **Type**: Unsigned Integer
  - **Description**: The number of the maximum block timestamp jump the fuzzer should make between test transactions. The fuzzer will use this value to make the next block's timestamp between `[1, blockTimestampDelayMax]` more than that of the previous block. Jumping time allows `medusa` to enter code paths that require a given amount of time to pass.
  - **Default**: 604_800
- `blockGasLimit`
  - **Type**: Unsigned Integer
  - **Description**: The number of the maximum amount of gas a block's transactions can use in total (thus defining max transactions per block).
    > 🚩 It is advised not to change this naively, as a minimum must be set for the chain to operate.
  - **Default**: 125_000_000
- `transactionGasLimit`
  - **Type**: Unsigned Integer
  - **Description**: Defines the amount of gas sent with each fuzzer-generated transaction.
    > 🚩 It is advised not to change this naively, as a minimum must be set for the chain to operate.
  - **Default**: 12_500_000
- `testing`
  - **Type**: Struct
  - **Description**: This struct will define the configuration for the various testing modes that are supported by `medusa`
  - **Components**:
    - `stopOnFailedTest`
      - **Type**: Boolean
      - **Description**: Determines whether the fuzzer should stop execution after the first _failed_ test.
      - **Default**: `true`
    - `stopOnFailedContractMatching`
      - **Type**: Boolean
      - **Description**: Determines whether the fuzzer should stop execution if it is unable to match the bytecode of a dynamically deployed contract. A dynamically deployed contract is one that is created during the fuzzing campaign (versus one that is specified in the `deploymentOrder`).
      - **Default**: `true`
    - `stopOnNoTests`
      - **Type**: Boolean
      - **Description**: Determines whether the fuzzer should stop execution if no tests are found (property tests, assertion tests, optimization tests, or custom API-level tests)
      - **Default**: `true`
    - `testAllContracts`
      - **Type**: Boolean
      - **Description**: Determines whether all contracts should be tested (including dynamically deployed ones), rather than just the contracts specified in the project configuration's `deploymentOrder`.
      - **Default**: `false`
    - `traceAll`:
      - **Type**: Boolean
      - **Description**: Determines whether an "execution trace" should be attached to each element of a call sequence that triggered a test failure. An execution trace showcases the various contract calls, return data, and emitted events that a call encountered. By default, a trace is attached for the _final_ call in the call sequence that triggered a test failure.
      - **Default**: `false`
    - `assertionTesting`
      - **Type**: Struct
      - **Description**: This struct describes the configuration for assertion testing mode.
      - **Components**:
        - `enabled`
          - **Type**: Boolean
          - **Description**: Enable or disable assertion-based testing
          - **Default**: `false`
        - `testViewMethods`
          - **Type**: Boolean
          - **Description**: Whether `pure` / `view` functions should be tested for assertion failures.
          - **Default**: `false`
        - `assertionModes`
          - **Type**: Struct
          - **Description**: This struct describes the various types of EVM-level panics that should be considered a "failing case".
          - **Components**:
            - `failOnCOmpilerInsertedPanic`
              - **Type**: Boolean
              - **Description**: Triggering a compiler-inserted panic should be treated as a failing case.
              - **Default**: `false`
            - `failOnAssertion`
              - **Type**: Boolean
              - **Description**: Triggering an assertion failure (e.g. `assert(false)`) should be treated as a failing case.
              - **Default**: `true`
            - `failOnArithmeticUnderflow`
              - **Type**: Boolean
              - **Description**: Arithmetic underflows or overflows should be treated as a failing case
              - **Default**: `false`
            - `failOnDivideByZero`
              - **Type**: Boolean
              - **Description**: Dividing by zero should be treated as a failing case
              - **Default**: `false`
            - `failOnEnumTypeConversionOutOfBounds`
              - **Type**: Boolean
              - **Description**: An out-of-bounds enum access should be treated as a failing case
              - **Default**: `false`
            - `failOnIncorrectStorageAccess`
              - **Type**: Boolean
              - **Description**: An out-of-bounds storage access should be treated as a failing case
              - **Default**: `false`
            - `failOnPopEmptyArray`
              - **Type**: Boolean
              - **Description**: A `pop` operation on an empty array should be treated as a failing case
              - **Default**: `false`
            - `failOnOutOfBoundsArrayAccess`
              - **Type**: Boolean
              - **Description**: An out-of-bounds array access should be treated as a failing case
              - **Default**: `false`
            - `failOnAllocateTooMuchMemory`
              - **Type**: Boolean
              - **Description**: Overallocation/excessive memory usage should be treated as a failing case
              - **Default**: `false`
            - `failOnCallUninitializedVariable`
              - **Type**: Boolean
              - **Description**: Calling an unitialized variable should be treated as a failing case
              - **Default**: `false`
    - `propertyTesting`
      - **Type**: Struct
      - **Description**: This struct describes the configuration for property testing mode.
      - **Components**:
        - `enabled`
          - **Type**: Boolean
          - **Description**: Enable or disable property-based testing.
          - **Default**: `true`
        - `testPrefixes`
          - **Type**: [String]
          - **Description**: The list of prefixes that the fuzzer will use to determine whether a given function is a property test or not. For example, if `fuzz_` is a test prefix, then any function name in the form `fuzz_*` may be a property test.
            > **Note**: If you are moving over from Echidna, you can add `echidna_` as a test prefix to quickly port over the property tests from it.
          - **Default**: `[fuzz_]`
    - `optimizationTesting`
      - **Type**: Struct
      - **Description**: This struct describes the configuration for optimization testing mode.
      - **Components**:
        - `enabled`
          - **Type**: Boolean
          - **Description**: Enable or disable optimization testing.
          - **Default**: `true`
        - `testPrefixes`
          - **Type**: [String]
          - **Description**: The list of prefixes that the fuzzer will use to determine whether a given function is an optimization test or not. For example, if `optimize_` is a test prefix, then any function name in the form `optimize_*` may be a property test.
          - **Default**: `[optimize_]`

## Compilation Configuration

The compilation configuration defines the parameters to use while compiling a target file or project:

- `platform`
  - **Type**: String
  - **Description**: Refers to the type of platform to be used to compile the underlying target. Currently, `crytic-compile` or `solc` can be used as the compilation platform.
- `platformConfig`
  - **Type**: Struct
  - **Description**: This struct is a platform-dependent structure which offers parameters for compiling the underlying project. See below for the structure of `platformConfig` for each compilation platform

> **Note**: The `target` parameter in each `platformConfig` below must be _relative_ paths from the directory containing `medusa`'s project configuration file.

### `platformConfig` for each platform

#### `crytic-compile`

- `target`
  - **Type**: String
  - **Description**: Refers to the target that is being compiled. The target can be a single `.sol` file or a whole project (e.g. `hardhat` / `foundry` project).
- `solcVersion`
  - **Type**: String
  - **Description**: Describes the version of `solc` that will be installed and then used for compilation.
- `exportDirectory`
  - **Type**: String
  - **Description**: Describes the directory where all compilation artifacts will be stored.
- `args`
  - **Type**: [String]
  - **Description**: Refers to any additional args that one may want to provide to `crytic-compile`.
    > 🚩 The `--export-format` and `--export-dir` are already used during compilation with `crytic-compile`. Re-using these flags in `args` will cause the compilation to fail.

#### `solc`

- `target`
  - **Type**: String
  - **Description**: Refers to the target that is being compiled. The target must be a single `.sol` file.

## Logging Configuration

The logging configuration defines the parameters for logging to console and/or file:

- `level`
  - **Type**: String
  - **Description**: The log level will determine which logs are emitted or discarded. If `level` is "info" then all logs with informational severity or higher will be logged.
  - **Default**: "info"
- `logDirectory`
  - **Type**: String
  - **Description**: Describes what directory log files should be outputted. Have a non-empty `logDirectory` value will enable "file logging" which will result in logs to be output to both console and file. Note that the directory path is _relative_ to the directory containing `medusa`'s project configuration file.
