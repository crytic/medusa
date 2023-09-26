# Fuzzing Configuration

The fuzzing configuration defines the parameters for the fuzzing campaign.

### `workers`
- **Type**: Integer
- **Description**: The number of worker threads to parallelize fuzzing operations on.
- **Default**: 10 workers

### `workerResetLimit`
- **Type**: Integer
- **Description**: The number of call sequences a worker should process on its underlying chain before being fully reset,
freeing memory. After resetting, the worker will be re-created and continue processing of call sequences.
  > 🚩 This setting, along with `workers` influence the speed and memory consumption of the fuzzer. Setting this value 
  > higher will result in greater memory consumption per worker. Setting it too high will result in the in-memory 
  > chain's database growing to a size that is slower to process. Setting it too low may result in frequent worker resets
  > that are computationally expensive for complex contract deployments that need to be replayed during worker reconstruction.
- **Default**: 50 sequences

### `timeout`
- **Type**: Integer
- **Description**: The number of seconds before the fuzzing campaign should be terminated. If a zero value is provided,
the timeout will not be enforced. The timeout begins counting after compilation succeeds and the fuzzing campaign has started.
- **Default**: 0 seconds

### `testLimit`
- **Type**: Unsigned Integer
- **Description**: The number of function calls to make before the fuzzing campaign should be terminated. If a zero value
is provided, no call limit will be enforced.
- **Default**: 0 calls

### `callSequenceLength`
- **Type**: Integer
- **Description**: The maximum number of function calls to generate in a single call sequence in the attempt to violate 
properties. After every `callSequenceLength` function calls, the blockchain is reset for the next sequence of transactions.
- **Default**: 100 calls/sequence

### `coverageEnabled`
- **Type**: Boolean
- **Description**: Whether coverage-increasing call sequences should be saved in the corpus for the fuzzer to mutate/re-use.
Enabling coverage allows for improved code exploration.
- **Default**: `true`

### `corpusDirectory`
- **Type**: String
- **Description**: The file path where the corpus should be saved. The corpus collects sequences during a fuzzing campaign
that help drive fuzzer features (e.g. a call sequence that increases code coverage is stored in the corpus). These sequences
can then be re-used/mutated by the fuzzer during the next fuzzing campaign.
- **Default**: ""

### `deploymentOrder`
- **Type**: [String] (e.g. `[FirstContract, SecondContract, ThirdContract]`)
- **Description**: The list of contracts that will be deployed on the blockchain and then targeted for fuzzing by medusa.
For single-contract compilations, this value can be left as `[]`. This, however, is rare since most projects are multi-contract compilations. 
> 🚩 Note that the order specified in the array is the _order_ in which the contracts are deployed to the blockchain. 
> Thus, if you have a `corpusDirectory` set up, and you change the order of the contracts in the array, the corpus may no
> longer work since the contract addresses of the target contracts will change.
- **Default**: `[]`

### `constructorArgs`
- **Type**: `{"contractName": {"variableName": _value}}`
- **Description**: If a contract in the `deploymentOrder` has a constructor that takes in variables, these can be specified here. 
An example can be found [here](TODO).
- **Default**: `{}`

### `deployerAddress`
- **Type**: Address
- **Description**: The address used to deploy contracts on startup, represented as a hex string.
> 🚩 Changing this address may render entries in the corpus invalid since the addresses of the target contracts will change.
> It is recommended to clear your corpus when doing so.
- **Default**: `0x30000`

### `senderAddresses`
- **Type**: [Address]
- **Description**: Defines the account addresses used to send function calls to deployed contracts in the fuzzing campaign.
> 🚩 Changing these addresses may render entries in the corpus invalid since the sender(s) of corpus transactions may no
> longer be valid. It is recommended to clear your corpus when doing so.
- **Default**: `[0x10000, 0x20000, 0x30000]`

### `blockNumberDelayMax`
- **Type**: Integer
- **Description**: Defines the maximum block number jump the fuzzer should make between test transactions. The fuzzer 
will use this value to make the next block's `block.number` between `[1, blockNumberDelayMax]` more than that of the previous
block. Jumping `block.number` allows medusa to enter code paths that require a given number of blocks to pass.
- **Default**: 60_480

### `blockTimestampDelayMax`
- **Type**: Integer
- **Description**: The number of the maximum block timestamp jump the fuzzer should make between test transactions. 
The fuzzer will use this value to make the next block's `block.timestamp` between `[1, blockTimestampDelayMax]` more 
than that of the previous block. Jumping `block.timestamp`time allows medusa to enter code paths that require a given amount of time to pass.
- **Default**: 604_800

### `blockGasLimit`
- **Type**: Integer
- **Description**: The maximum amount of gas a block's transactions can use in total (thus defining max transactions per block).
> 🚩 It is advised not to change this naively, as a minimum must be set for the chain to operate.
- **Default**: 125_000_000

### `transactionGasLimit`
- **Type**: Integer
- **Description**: Defines the amount of gas sent with each fuzzer-generated transaction.
> 🚩 It is advised not to change this naively, as a minimum must be set for the chain to operate.
- **Default**: 12_500_000
