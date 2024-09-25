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
  the timeout will not be enforced. The timeout begins after compilation succeeds and the fuzzing campaign has started.
- **Default**: 0 seconds

### `testLimit`

- **Type**: Integer
- **Description**: The number of function calls to make before the fuzzing campaign should be terminated. If a zero value
  is provided, no test limit will be enforced.
- **Default**: 0 calls

### `callSequenceLength`

- **Type**: Integer
- **Description**: The maximum number of function calls to generate in a single call sequence in the attempt to violate
  properties. After every `callSequenceLength` function calls, the blockchain is reset for the next sequence of transactions.
- **Default**: 100 calls/sequence

### `coverageEnabled`

- **Type**: Boolean
- **Description**: Whether coverage-increasing call sequences should be saved for the fuzzer to mutate/re-use.
  Enabling coverage allows for improved code exploration.
- **Default**: `true`

### `corpusDirectory`

- **Type**: String
- **Description**: The file path where the corpus should be saved. The corpus collects sequences during a fuzzing campaign
  that help drive fuzzer features (e.g. a call sequence that increases code coverage is stored in the corpus). These sequences
  can then be re-used/mutated by the fuzzer during the next fuzzing campaign.
- **Default**: ""

### `coverageFormats`

- **Type**: [String] (e.g. `["lcov"]`)
- **Description**: The coverage reports to generate after the fuzzing campaign has completed. The coverage reports are saved
  in the `coverage` directory within `crytic-export/` or `corpusDirectory` if configured.
- **Default**: `["lcov", "html"]`

### `targetContracts`

- **Type**: [String] (e.g. `[FirstContract, SecondContract, ThirdContract]`)
- **Description**: The list of contracts that will be deployed on the blockchain and then targeted for fuzzing by `medusa`.
  For single-contract compilations, this value can be left as `[]`. This, however, is rare since most projects are multi-contract compilations.
  > 🚩 Note that the order specified in the array is the _order_ in which the contracts are deployed to the blockchain.
  > Thus, if you have a `corpusDirectory` set up, and you change the order of the contracts in the array, the corpus may no
  > longer work since the contract addresses of the target contracts will change. This may render the entire corpus useless.
- **Default**: `[]`

### `predeployedContracts`

- **Type**: `{"contractName": "contractAddress"}` (e.g.`{"TestContract": "0x1234"}`)
- **Description**: This configuration parameter allows you to deterministically deploy contracts at predefined addresses.
  > 🚩 Predeployed contracts do not accept constructor arguments. This may be added in the future.
- **Default**: `{}`

### `targetContractBalances`

- **Type**: [Base-16 Strings] (e.g. `[0x123, 0x456, 0x789]`)
- **Description**: The starting balance for each contract in `targetContracts`. If the `constructor` for a target contract
  is marked `payable`, this configuration option can be used to send ether during contract deployment. Note that this array
  has a one-to-one mapping to `targetContracts`. Thus, if `targetContracts` is `[A, B, C]` and `targetContractsBalances` is
  `["0", "0xff", "0"]`, then `B` will have a starting balance of 255 wei and `A` and `C` will have zero wei. Note that the wei-value
  has to be hex-encoded and _cannot_ have leading zeros. For an improved user-experience, the balances may be encoded as base-10
  format strings in the future.
- **Default**: `[]`

### `constructorArgs`

- **Type**: `{"contractName": {"variableName": _value}}`
- **Description**: If a contract in the `targetContracts` has a `constructor` that takes in variables, these can be specified here.
  An example can be found [here](#using-constructorargs).
- **Default**: `{}`

### `deployerAddress`

- **Type**: Address
- **Description**: The address used to deploy contracts on startup, represented as a hex string.
  > 🚩 Changing this address may render entries in the corpus invalid since the addresses of the target contracts will change.
- **Default**: `0x30000`

### `senderAddresses`

- **Type**: [Address]
- **Description**: Defines the account addresses used to send function calls to deployed contracts in the fuzzing campaign.
  > 🚩 Changing these addresses may render entries in the corpus invalid since the sender(s) of corpus transactions may no
  > longer be valid.
- **Default**: `[0x10000, 0x20000, 0x30000]`

### `blockNumberDelayMax`

- **Type**: Integer
- **Description**: Defines the maximum block number jump the fuzzer should make between test transactions. The fuzzer
  will use this value to make the next block's `block.number` between `[1, blockNumberDelayMax]` more than that of the previous
  block. Jumping `block.number` allows `medusa` to enter code paths that require a given number of blocks to pass.
- **Default**: `60_480`

### `blockTimestampDelayMax`

- **Type**: Integer
- **Description**: The number of the maximum block timestamp jump the fuzzer should make between test transactions.
  The fuzzer will use this value to make the next block's `block.timestamp` between `[1, blockTimestampDelayMax]` more
  than that of the previous block. Jumping `block.timestamp`time allows `medusa` to enter code paths that require a given amount of time to pass.
- **Default**: `604_800`

### `blockGasLimit`

- **Type**: Integer
- **Description**: The maximum amount of gas a block's transactions can use in total (thus defining max transactions per block).
  > 🚩 It is advised not to change this naively, as a minimum must be set for the chain to operate.
- **Default**: `125_000_000`

### `transactionGasLimit`

- **Type**: Integer
- **Description**: Defines the amount of gas sent with each fuzzer-generated transaction.
  > 🚩 It is advised not to change this naively, as a minimum must be set for the chain to operate.
- **Default**: `12_500_000`

## Using `constructorArgs`

There might be use cases where contracts in `targetContracts` have constructors that accept arguments. The `constructorArgs`
configuration option allows you to specify those arguments. `constructorArgs` is a nested dictionary that maps
contract name -> variable name -> variable value. Let's look at an example below:

```solidity
// This contract is used to test deployment of contracts with constructor arguments.
contract TestContract {
    struct Abc {
        uint a;
        bytes b;
    }

    uint x;
    bytes2 y;
    Abc z;

    constructor(uint _x, bytes2 _y, Abc memory _z) {
        x = _x;
        y = _y;
        z = _z;
    }
}

contract DependentOnTestContract {
    address deployed;

    constructor(address _deployed) {
        deployed = _deployed;
    }
}
```

In the example above, we have two contracts `TestContract` and `DependentOnTestContract`. You will note that
`DependentOnTestContract` requires the deployment of `TestContract` _first_ so that it can accept the address of where
`TestContract` was deployed. On the other hand, `TestContract` requires `_x`, `_y`, and `_z`. Here is what the
`constructorArgs` value would look like for the above deployment:

> **Note**: The example below has removed all the other project configuration options outside of `targetContracts` and
> `constructorArgs`

```json
{
  "fuzzing": {
    "targetContracts": ["TestContract", "DependentOnTestContract"],
    "constructorArgs": {
      "TestContract": {
        "_x": "123456789",
        "_y": "0x5465",
        "_z": {
          "a": "0x4d2",
          "b": "0x54657374206465706c6f796d656e74207769746820617267756d656e7473"
        }
      },
      "DependentOnTestContract": {
        "_deployed": "DeployedContract:TestContract"
      }
    }
  }
}
```

First, let us look at `targetContracts`. As mentioned in the [documentation for `targetContracts`](#targetcontracts),
the order of the contracts in the array determine the order of deployment. This means that `TestContract` will be
deployed first, which is what we want.

Now, let us look at `constructorArgs`. `TestContract`'s dictionary specifies the _exact name_ of the constructor argument
(e.g. `_x` or `_y`) with their associated value. Since `_z` is of type `TestContract.Abc`, `_z` is also a dictionary
that specifies each field in the `TestContract.Abc` struct.

For `DependentOnTestContract`, the `_deployed` key has
a value of `DeployedContract:TestContract`. This tells `medusa` to look for a deployed contract that has the name
`TestContract` and provide its address as the value for `_deployed`. Thus, whenever you need a deployed contract's
address as an argument for another contract, you must follow the format `DeployedContract:<ContractName>`.
