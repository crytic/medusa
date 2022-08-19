# medusa
`medusa` is a cross-platform [go-ethereum](https://github.com/ethereum/go-ethereum/)-based smart contract fuzzer inspired by [echidna](https://github.com/crytic/echidna).

## Requirements
- To use `medusa` with different compilation platforms such as `solc`, `truffle` or `crytic-compile`, you must ensure the relevant software is installed and available through your system's `PATH` environment variable (accessible from command-line in any working directory). 
- Building from source on Windows, the `go-ethereum` dependency may require [TDM-GCC](https://stackoverflow.com/questions/43580131/exec-gcc-executable-file-not-found-in-path-when-trying-go-build) to build.

## Usage
`medusa` is configuration file driven. Command-line arguments simply specify where to initialize a new medusa project configuration or where to ingest one.

### Initializing a project configuration
To create a `medusa` project configuration, invoke `medusa init [platform]` to create a configuration for the provided platform within your current working directory. Invoking this command without a supported `platform` argument will present you a list of supported platforms.

### Understanding the project configuration format
After initializing a `medusa` project in a given directory, a `medusa.json` file will be created. This is the project configuration file which dictates compilation and fuzzing parameters for `medusa`.

While the majority of the configuration structure is consistent, the `platform` which your project targets will offer differing compilation parameters. This means a project configuration for a `truffle` project will differ from one using `solc`.

An example project configuration can be observed below:
```json
{
	"accounts": {
		"generate": 5,
		"keys": [
			"9E9E0488FA54525AE25A5277BF3D3EF8FC44E0686210005EA10C7F07C69FCE28",
			"B3C0A41BC2AB9EB9239C006D9844FDF32E03F947D969933FD4503793F0C7C9D7"
		]
	},
	"fuzzing": {
		"workers": 20,
		"worker_database_entry_limit": 20000,
		"timeout": 0,
		"test_limit": 0,
		"max_tx_sequence_length": 10,
		"test_prefixes": [
			"fuzz_"
		]
	},
	"compilation": {
		"platform": "truffle",
		"platform_config": {
			"target": ".",
			"use_npx": true
		}
	}
}
```

The structure is described below:
- `accounts` defines which accounts will be used by `medusa` in a fuzzing campaign:
  - `generate` specifies the number of accounts to create and fund. 
  - `keys` can be used to specify an array of hex strings which will be interpreted as pre-defined account private keys to use in the fuzzing campaign.
- `fuzzing` defines parameters for the fuzzing campaign:
  - `workers` defines the number of worker threads to parallelize fuzzing operations on.
  - `worker_database_entry_limit` defines how many keys a worker's memory database can contain before the worker is reset
    - **Note**: this is a temporary logic for memory throttling
  - `timeout` refers to the number of seconds before the fuzzing campaign should be terminated. If a zero value is provided, the timeout will not be enforced.
  - `test_limit` refers to a threshold of the number of transactions to run before the fuzzing campaign should be terminated. Must be a non-negative number. If a zero value is provided, no transaction limit will be enforced.
  - `max_tx_sequence_length` defines the maximum number of transactions to generate in a single sequence that tries to violate property tests. For property tests which require many transactions to violate, this number should be set sufficiently high.
  - `test_prefixes` defines the list of prefixes that medusa will use to determine whether a given function is a property test or not. For example, if `fuzz_` is a test prefix, then any function name in the form `fuzz_*` may be a property test. There must be _at least_ one default test prefix. Note that if you are using Echidna, you can add `echidna_` as a test prefix to quickly port over the property tests from it.
- `compilation` defines parameters used to compile a given target to be fuzzed:
  - `platform` refers to the type of platform to be used to compile the underlying target.
  - `platform_config` is a platform-dependent structure which offers parameters for compiling the underlying project. Target paths are relative to the directory containing the `medusa` project configuration file.

### Writing property tests
Property tests are represented as functions within a Solidity contract whose names are prefixed with a prefix specified by the `test_prefixes` configuration option (`fuzz_` is the default test prefix). Additionally, they must take no arguments and return a `bool` indicating if the test succeeded.
```solidity
contract TestXY {
    uint x;
    uint y;

    function setX(uint value) public {
        x = value + 3;
    }

    function setY(uint value) public {
        y = value + 9;
    }

    function fuzz_never_specific_values() public view returns (bool) {
        // ASSERTION: x should never be 10 at the same time y is 80
        return !(x == 10 && y == 80);
    }
}
```
`medusa` deploys your contract containing property tests and generates a sequence of transactions to execute against all publicly accessible methods. After each transaction, it calls upon your property tests to ensure they return a `true` (success) status.

**Note**: `medusa` only deploys smart contracts which do not take arguments in their constructor. If you wish to test a contract which takes arguments, create a contract which inherits from it and satisfies the constructor arguments.


### Fuzzing with an existing project configuration
To begin a fuzzing campaign, invoke `medusa fuzz` to compile and fuzz targets specified by a `medusa` project configuration in your current working directory, or invoke `medusa fuzz [config_path]` to specify a path to a configuration outside the current working directory.

Invoking a fuzzing campaign, `medusa` will:
- Compile the given targets
- Start the configured number of worker threads, each with their own local Ethereum test node.
- Deploy all contracts which contain no constructor arguments to each worker's test node.
- Begin to generate and send transaction sequences to update contract state.
- Check property tests all succeed after each transaction executed.

Upon discovery of a failed property test, `medusa` will halt, reporting the transaction sequence used to violate any property test(s):
```
Failed property tests: fuzz_never_specific_values()
Transaction Sequence:
[1] setX([7]) (sender=0x73bAF44082D78657f204ABE15E5A00bAC56820aD, gas=4712388, gasprice=1000000000, value=0)
[2] setY([71]) (sender=0x857371A82Cc85dA5CCcd68E4dffC6A5248c659b0, gas=4712388, gasprice=1000000000, value=0)
```

## Contributing
For information about how to contribute to this project, check out the [CONTRIBUTING](./CONTRIBUTING.md) guideline.
