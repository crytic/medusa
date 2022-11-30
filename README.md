# medusa
`medusa` is a cross-platform [go-ethereum](https://github.com/ethereum/go-ethereum/)-based smart contract fuzzer inspired by [echidna](https://github.com/crytic/echidna).

## Requirements
- To use `medusa` with different compilation platforms such as `solc`, `truffle` or `crytic-compile`, you must ensure the relevant software is installed and available through your system's `PATH` environment variable (accessible from command-line in any working directory). 
- Building from source on Windows, the `go-ethereum` dependency may require [TDM-GCC](https://stackoverflow.com/questions/43580131/exec-gcc-executable-file-not-found-in-path-when-trying-go-build) to build.

## Usage
`medusa` can be configured both using a configuration file and with command line arguments.

### Initializing a project configuration
To create a `medusa` project configuration, invoke `medusa init [platform]` to create a configuration for the provided platform within your current working directory. Invoking this command without a `platform` argument will result in using `crytic-compile` as the default compilation platform. 

> Note that the output of `medusa init`, which is equivalent to `medusa init crytic-compile`, is considered the **default configuration** of `medusa`

### Using the CLI to modify project configuration initialization
While running `medusa init`, you have access to two flags:
1. `medusa init [platform] --out myConfig.json`: The `--out` value will determine the output path where your project configuration will live. Without `--out`, the default output path is `medusa.json` in the current working directory.
2. `medusa init [platform] --target myContract.sol`: The `--target` value will determine the compilation target which can be a single file or project.

### Understanding the project configuration format
After initializing a `medusa` project in a given directory by running `medusa init [platform]`, a `medusa.json` file will be created (or a custom one if `--out` is used). This is the project configuration file which dictates compilation and fuzzing parameters for `medusa`.

While the majority of the configuration structure is consistent, the `platform` which your project targets will offer differing compilation parameters. This means a project configuration for a `truffle` project will differ from one using `solc`.

An example project configuration can be observed below:
```json
{
	"fuzzing": {
		"workers": 10,
		"workerResetLimit": 50,
		"timeout": 0,
		"testLimit": 10000000,
		"callSequenceLength": 100,
		"corpusDirectory": "corpus",
		"coverageEnabled": true,
		"deploymentOrder": ["TestXY", "TestContract2"],
		"deployerAddress": "0x1111111111111111111111111111111111111111",
		"senderAddresses": [
			"0x1111111111111111111111111111111111111111",
			"0x2222222222222222222222222222222222222222",
			"0x3333333333333333333333333333333333333333"
		],
		"blockNumberDelayMax": 60480,
		"blockTimestampDelayMax": 604800,
		"blockGasLimit": 12500000,
		"transactionGasLimit": 12500000,
		"testing": {
			"stopOnFailedTest": true,
			"assertionTesting": {
				"enabled": false,
				"testViewMethods": false
			},
			"propertyTesting": {
				"enabled": true,
				"testPrefixes": [
					"fuzz_"
				]
			}
		}
	},
	"compilation": {
		"platform": "crytic-compile",
		"platformConfig": {
			"target": ".",
			"args": []
		}
	}
}
```

The structure is described below:
- `fuzzing` defines parameters for the fuzzing campaign:
  - `workers` defines the number of worker threads to parallelize fuzzing operations on.
  - `workerResetLimit` defines how many call sequences a worker should process on its underlying chain before being fully reset, freeing memory.
    - **Note**: This setting, along with `workers` influence the speed and memory consumption of the fuzzer. Setting this value higher will result in greater memory consumption per worker. Setting it too high will result in the in-memory chain's database growing to a size that is slower to process. Setting it too low may result in frequent worker resets that are computationally expensive for complex contract deployments that need to be replayed during worker reconstruction.
  - `timeout` refers to the number of seconds before the fuzzing campaign should be terminated. If a zero value is provided, the timeout will not be enforced. The timeout begins counting after compilation succeeds and the fuzzing campaign is starting.
  - `testLimit` refers to a threshold of the number of function calls to make before the fuzzing campaign should be terminated. Must be a non-negative number. If a zero value is provided, no call limit will be enforced.
  - `callSequenceLength` defines the maximum number of function calls to generate in a single sequence that tries to violate property tests. For property tests which require many calls to violate, this number should be set sufficiently high.
  - `corpusDirectory` refers to the path where the corpus should be saved. The corpus collects artifacts during a fuzzing campaign that help drive fuzzer features (e.g. coverage-increasing call sequences which the fuzzer collects to be mutates).
  - `coverageEnabled` refers to whether coverage-increasing call sequences should be saved in the corpus for the fuzzer to mutate. This aims to help achieve greater coverage across contracts when testing.
  - `deploymentOrder` refers to the order in which compiled contracts (contracts resulting from compilation, as specified by the `compilation` config) should be deployed to the fuzzer on startup. At least one contract name must be specified here. 
    - **Note**: Changing this order may render entries in the corpus invalid. It is recommended to clear your corpus when doing so.
  - `deployerAddress` defines the account address used to deploy contracts on startup, represented as a hex string.
      - **Note**: Changing this address may render entries in the corpus invalid. It is recommended to clear your corpus when doing so.
  - `senderAddresses` defines the account addresses used to send function calls to deployed contracts in the fuzzing campaign.
      - **Note**: Removing previously existing addresses may render entries in the corpus invalid. It is recommended to clear your corpus when doing so.
  - `blockNumberDelayMax` defines the maximum block number jump the fuzzer should make between test transactions.
  - `blockTimestampDelayMax` defines the maximum block timestamp jump the fuzzer should make between test transactions.
  - `blockGasLimit` defines the maximum amount of gas a block's transactions can use in total (thus defining max transactions per block). 
    - **Note**: It is advised not to change this naively, as a minimum must be set for the chain to operate.
  - `transactionGasLimit` defines the amount of gas sent with each fuzzer-generated transaction.
    - **Note**: It is advised not to change this naively, as a minimum must be set for the chain to operate.
  - `testing` defines the configuration for built-in test case providers which can be leveraged for fuzzing campaign.
    - `stopOnFailedTest` defines whether the fuzzer should exit after detecting the first failed test, or continue fuzzing to find other results.
    - `assertionTesting` describes configuration for assertion-based testing.
      - `enabled` describes whether assertion testing should be enabled.
      - `testViewMethods` describes whether pure/view functions should be tested for assertion failures.
    - `propertyTesting` describes configuration for property-based testing.
      - `enabled` describes whether property testing should be enabled.
      - `testPrefixes` defines the list of prefixes that medusa will use to determine whether a given function is a property test or not. For example, if `fuzz_` is a test prefix, then any function name in the form `fuzz_*` may be a property test. Note that if you are using Echidna, you can add `echidna_` as a test prefix to quickly port over the property tests from it.
- `compilation` defines parameters used to compile a given target to be fuzzed:
  - `platform` refers to the type of platform to be used to compile the underlying target.
  - `platformConfig` is a platform-dependent structure which offers parameters for compiling the underlying project. Target paths are relative to the directory containing the `medusa` project configuration file.

### Using the CLI to update project configuration
In addition to using a project configuration file to provide the necessary parameters to `medusa`, you can also use `medusa`'s CLI. 
Note that the CLI flags do not provide the same level of control to `medusa`'s system configuration as the configuration file does. Thus, the CLI flags and the configuration file can be used in tandem. This results in four distinct possibilities:
1. **Configuration file with no CLI flags**: This will result in `medusa` retrieving all of its project configuration parameters directly from the file.
2. **Configuration file with CLI flags**: This will result in `medusa` using the file as the base configuration and using the CLI arguments to override specific configuration items.
3. **No configuration file and CLI flags**: This is equivalent to `medusa` using its **default configuration** (i.e. output of `medusa init`) and using the CLI arguments to override specific configuration items.
4. **No configuration file and no CLI flags**: This is equivalent to `medusa` using its **default configuration**.

Currently, the following flags can be used with the `medusa fuzz` command to update the configuration.
- `medusa fuzz --config myConfig.json`: Will use the configuration in `myConfig.json` as the project configuration. If `--config` is not set, `medusa` will look for a `medusa.json` file in the current working directory. 
- `medusa fuzz --target myContract.sol`: Will set the compilation target to `myContract.sol`
- `medusa fuzz --workers 20`: Will set the number of `workers` to 20 threads
- `medusa fuzz --timeout 1000`: Will set the `timeout` to 1000 seconds
- `medusa fuzz --test-limit 50000`: Will set the `testLimit` to 50000 function calls
- `medusa fuzz --seq-len 50`: Will set the `maxTxSequenceLength` to 50 transactions
- `medusa fuzz --deployment-order "FirstContract,SecondContract"`: Will set the deployment order to `[FirstContract, SecondContract]`
- `medusa fuzz --corpus-dir myCorpus`: Will set the corpus directory _path_ to `myCorpus`
- `medusa fuzz --senders "0x10000,0x20000,0x30000"`: Will set the `senderAdddresses` to `[0x10000, 0x20000, 0x30000]`
- `medusa fuzz --deployer "0x10000"`: Will set the `deployerAddress` to `0x10000`
- `medusa fuzz --assertion-mode`: Will set `assertionTesting.enabled` to `true`

### Running medusa

We have a variety of different ways of running `medusa`:
1. `medusa fuzz`: Run `medusa` using the configuration in `medusa.json` (or the **default configuration** if `medusa.json` can't be found) and no CLI updates
4. `medusa fuzz --workers 20 --test-limit 50000`: Run `medusa` using the configuration in `medusa.json` (or the **default configuration** if `medusa.json` can't be found) and then override the `workers` and `testLimit` parameters
2. `medusa fuzz --config myConfig.json`: Run `medusa` using the configuration in `myConfig.json` (or the **default configuration** if `myConfig.json` can't be found) and no CLI updates
3. `medusa fuzz --config myConfig.json --workers 20 --test-limit 50000`: Run `medusa` using the configuration in `myConfig.json` (or the **default configuration** if `myConfig.json` can't be found) and then override the `workers` and `testLimit` parameters.

### Writing property tests
Property tests are represented as functions within a Solidity contract whose names are prefixed with a prefix specified by the `testPrefixes` configuration option (`fuzz_` is the default test prefix). Additionally, they must take no arguments and return a `bool` indicating if the test succeeded.
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
`medusa` deploys your contract containing property tests and generates a sequence of calls to execute against all publicly accessible methods. After each function call, it calls upon your property tests to ensure they return a `true` (success) status.

**Note**: `medusa` only deploys smart contracts which do not take arguments in their constructor. If you wish to test a contract which takes arguments, create a contract which inherits from it and satisfies the constructor arguments.


### Fuzzing with an existing project configuration
To begin a fuzzing campaign, invoke `medusa fuzz` to compile and fuzz targets specified by a `medusa.json` project configuration in your current working directory, or invoke `medusa fuzz --config [config_path]` to specify a unique configuration file.

Invoking a fuzzing campaign, `medusa` will:
- Compile the given targets
- Start the configured number of worker threads, each with their own local Ethereum test chain.
- Deploy all contracts which contain no constructor arguments to each worker's test chain.
- Begin to generate and send call sequences to update contract state.
- Check property tests all succeed after each call executed.

Upon discovery of a failed property test, `medusa` will halt, reporting the call sequence used to violate any property test(s):
```
[FAILED] Property Test: TestXY.fuzz_never_specific_values()
Test "TestXY.fuzz_never_specific_values()" failed after the following call sequence:
1) TestXY.setY([71]) (gas=4712388, gasprice=1, value=0, sender=0x2222222222222222222222222222222222222222)
2) TestXY.setX([7]) (gas=4712388, gasprice=1, value=0, sender=0x3333333333333333333333333333333333333333)
```

## Contributing
For information about how to contribute to this project, check out the [CONTRIBUTING](./CONTRIBUTING.md) guideline.
