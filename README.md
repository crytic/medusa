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
	"fuzzing": {
		"workers": 10,
		"workerDatabaseEntryLimit": 10000,
		"timeout": 0,
		"testLimit": 10000000,
		"maxTxSequenceLength": 100,
		"corpusDirectory": "corpus",
		"coverageEnabled": true,
		"deploymentOrder": ["TestXY", "TestContract2"],
		"deployerAddress": "0x1111111111111111111111111111111111111111",
		"senderAddresses": [
			"0x1111111111111111111111111111111111111111",
			"0x2222222222222222222222222222222222222222",
			"0x3333333333333333333333333333333333333333"
		],
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
  - `workerDatabaseEntryLimit` defines how many keys a worker's memory database can contain before the worker is reset
    - **Note**: this is a temporary logic for memory throttling
  - `timeout` refers to the number of seconds before the fuzzing campaign should be terminated. If a zero value is provided, the timeout will not be enforced. The timeout begins counting after compilation succeeds and the fuzzing campaign is starting.
  - `testLimit` refers to a threshold of the number of function calls to make before the fuzzing campaign should be terminated. Must be a non-negative number. If a zero value is provided, no call limit will be enforced.
  - `maxTxSequenceLength` defines the maximum number of function calls to generate in a single sequence that tries to violate property tests. For property tests which require many calls to violate, this number should be set sufficiently high.
  - `corpusDirectory` refers to the path where the corpus should be saved. The corpus collects artifacts during a fuzzing campaign that help drive fuzzer features (e.g. coverage-increasing call sequences which the fuzzer collects to be mutates).
  - `coverageEnabled` refers to whether coverage-increasing call sequences should be saved in the corpus for the fuzzer to mutate. This aims to help achieve greater coverage across contracts when testing.
  - `deploymentOrder` refers to the order in which compiled contracts (contracts resulting from compilation, as specified by the `compilation` config) should be deployed to the fuzzer on startup. At least one contract name must be specified here. 
    - **Note**: Changing this order may render entries in the corpus invalid. It is recommended to clear your corpus when doing so.
  - `deployerAddress` defines the account address used to deploy contracts on startup, represented as a hex string.
      - **Note**: Changing this address may render entries in the corpus invalid. It is recommended to clear your corpus when doing so.
  - `senderAddresses` defines the account addresses used to send function calls to deployed contracts in the fuzzing campaign.
      - **Note**: Removing previously existing addresses may render entries in the corpus invalid. It is recommended to clear your corpus when doing so.
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
To begin a fuzzing campaign, invoke `medusa fuzz` to compile and fuzz targets specified by a `medusa` project configuration in your current working directory, or invoke `medusa fuzz [config_path]` to specify a path to a configuration outside the current working directory.

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
