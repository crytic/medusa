# medusa

`medusa` is a cross-platform [go-ethereum](https://github.com/ethereum/go-ethereum/)-based smart contract fuzzer inspired by [echidna](https://github.com/crytic/echidna). 
It provides parallelized fuzz testing of smart contracts through CLI, or its Go API that allows for custom user-extended testing methodology.

## Features

`medusa` provides support for:
- ✔️**Parallel fuzzing and testing** methodologies across multiple workers (threads)
- ✔️**Assertion and property testing**: built-in support for writing basic Solidity property tests and assertion tests
- ✔️**Mutational value generation**: fed by compilation and runtime values.
- ✔️**Coverage collecting**: Coverage increasing call sequences are stored in the corpus
- ❌ **Coverage guided**: Coverage increasing call sequences from the corpus are mutated to further guide the fuzzing campaign
- ✔️**Extensible low-level testing API** through events and hooks provided throughout the fuzzer, workers, and test chains.
- ❌ **Extensible high-level testing API** allowing for the addition of per-contract or global post call/event property tests


## Installation

### Precompiled binaries

To use `medusa`, first ensure you have [crytic-compile](https://github.com/crytic/crytic-compile) and a suitable compilation framework (e.g. `solc`, `truffle`, `hardhat`) installed on your machine.

You can then fetch the latest binaries for your platform from our [GitHub Releases](https://github.com/trailofbits/medusa) page.


### Building from source

#### Requirements

- You must have at least go 1.18 installed.
- [Windows only] The `go-ethereum` dependency may require [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) to build.

#### Steps

- Clone the repository, then execute `go build` in the repository root.
- Go will automatically fetch all dependencies and build a binary for you in the same folder when completed.


## Usage

`medusa` is primarily configuration file driven, though it can be run with a default config with CLI arguments that specify the target compilation path and contracts.

### Command-line Only

You can use the following command to run `medusa` against a contract:

```console
medusa fuzz --target contract.sol --deployment-order ContractName1,ContractName2
```

Where `--target` specifies the path `crytic-compile` should use to compile contracts, and `--deployment-order` specifies which contracts should be dpeloyed for testing.

### Configuration file driven

The preferred method to use medusa to enter your project directory (hardhat directory, or directory with your contracts),
then execute the following command:

```console
medusa init
```

This will create a `medusa.json` in your current folder. There are two required fields that should be set correctly:
- Set your `"target"` under `"compilation"` to point to the file/directory which `crytic-compile` should use to build your contracts.
- Put the names of any contracts you wish to deploy and run tests against in the `"deploymentOrder"` field. This must be non-empty.

After you have a configuration in place, you can execute:

```console
medusa fuzz
```

This will use the `medusa.json` configuration in the current directory and begin the fuzzing campaign.

Visit our [configuration]() wiki page for more information.


## Running Unit Tests

- Install [crytic-compile](https://github.com/crytic/crytic-compile), [solc-select](https://github.com/crytic/solc-select), and ensure you have `solc`, `truffle`, and `hardhat` available on your system.
- From the root of the project directory, invoke `go test -v ./...` on through command-line to run tests from all packages at or below the root.
  - Or enter each package directory to run `go test -v .` to test the immediate package.
  - Note: the `-v` parameter provides verbose output.
- Otherwise, use an IDE like [GoLand](https://www.jetbrains.com/go/) to visualize the tests and logically separate output.

## Contributing

For information about how to contribute to this project, check out the [CONTRIBUTING](./CONTRIBUTING.md) guideline.


## License

medusa is licensed and distributed under the [AGPLv3](./LICENSE).