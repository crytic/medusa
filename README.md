# medusa

`medusa` is a cross-platform [go-ethereum](https://github.com/ethereum/go-ethereum/)-based smart contract fuzzer inspired by [Echidna](https://github.com/crytic/echidna).
It provides parallelized fuzz testing of smart contracts through CLI, or its Go API that allows custom user-extended testing methodology.

**Disclaimer**: Please note that `medusa` is an **experimental** smart contract fuzzer. Currently, it should _not_ be adopted into production systems. We intend for `medusa` to reach the same capabilities and maturity that Echidna has. Until then, be careful using `medusa` as your primary smart contract fuzz testing solution. Additionally, please be aware that the Go-level testing API is still **under development** and is subject to breaking changes.

## Features

`medusa` provides support for:

- ✔️**Parallel fuzzing and testing** methodologies across multiple workers (threads)
- ✔️**Assertion and property testing**: built-in support for writing basic Solidity property tests and assertion tests
- ✔️**Mutational value generation**: fed by compilation and runtime values.
- ✔️**Coverage collecting**: Coverage increasing call sequences are stored in the corpus
- ✔️**Coverage guided fuzzing**: Coverage increasing call sequences from the corpus are mutated to further guide the fuzzing campaign
- ✔️**Extensible low-level testing API** through events and hooks provided throughout the fuzzer, workers, and test chains.
- ❌ **Extensible high-level testing API** allowing for the addition of per-contract or global post call/event property tests with minimal effort.

## Installation

### Precompiled binaries

To use `medusa`, ensure you have:

- [crytic-compile](https://github.com/crytic/crytic-compile) (`pip3 install crytic-compile`)
- a suitable compilation framework (e.g. `solc`, `hardhat`) installed on your machine. We recommend [solc-select](https://github.com/crytic/solc-select) to quickly switch between Solidity compiler versions.

You can then fetch the latest binaries for your platform from our [GitHub Releases](https://github.com/crytic/medusa/releases) page.

### Building from source

#### Requirements

- You must have at least go 1.18 installed.
- [Windows only] The `go-ethereum` dependency may require [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) to build.

#### Steps

- Clone the repository, then execute `go build` in the repository root.
- Go will automatically fetch all dependencies and build a binary for you in the same folder when completed.

## Usage

Although we recommend users run `medusa` in a configuration file driven format for more customizability, you can also run `medusa` through the CLI directly.
We provide instructions for both below.

We recommend you familiarize yourself with writing [assertion](https://github.com/crytic/building-secure-contracts/blob/master/program-analysis/echidna/basic/assertion-checking.md) and [property](https://github.com/crytic/building-secure-contracts/blob/master/program-analysis/echidna/introduction/how-to-test-a-property.md) tests for Echidna. `medusa` supports Echidna-like property testing and function optimization with config-defined function prefixes (default: `property_` and `optimize_`, respectively) and assertion testing using Solidity `assert(...)` statements.

### Command-line only

You can use the following command to run `medusa` against a contract:

```console
medusa fuzz --compilation-target contract.sol --target-contracts ContractName
```

Where:

- `--compilation-target` specifies the path `crytic-compile` should use to compile contracts
- `--target-contracts` specifies comma-separated names of contracts to be deployed for testing.

**Note:** Check out the [command-line interface](https://github.com/crytic/medusa/wiki/Command-Line-Interface) wiki page, or run `medusa --help` for more information.

### Configuration file driven

The preferred method to use medusa is to enter your project directory (hardhat directory, or directory with your contracts),
then execute the following command:

```console
medusa init
```

This will create a `medusa.json` in your current folder. There are two required fields that should be set correctly:

- Set your `"target"` under `"compilation"` to point to the file/directory which `crytic-compile` should use to build your contracts.
- Put the names of any contracts you wish to deploy and run tests against in the `"targetContracts"` field. This must be non-empty.

After you have a configuration in place, you can execute:

```console
medusa fuzz
```

This will use the `medusa.json` configuration in the current directory and begin the fuzzing campaign.

**Note:** Check out the [project configuration](https://github.com/crytic/medusa/wiki/Project-Configuration) wiki page, or run `medusa --help` for more information.

## Running Unit Tests

First, install [crytic-compile](https://github.com/crytic/crytic-compile), [solc-select](https://github.com/crytic/solc-select), and ensure you have `solc` (version >=0.8.7), and `hardhat` available on your system.

- From the root of the repository, invoke `go test -v ./...` on through command-line to run tests from all packages at or below the root.
  - Or enter each package directory to run `go test -v .` to test the immediate package.
  - Note: the `-v` parameter provides verbose output.
- Otherwise, use an IDE like [GoLand](https://www.jetbrains.com/go/) to visualize the tests and logically separate output.

## FAQs

**Why create `medusa` if Echidna is already working just fine?**

With `medusa`, we are exploring a different EVM implementation and language for our smart contract fuzzer. We believe that
experimenting with a new fuzzer provides us with the following benefits:

- Since `medusa` is written in Go, we believe that this will **lower the barrier of entry for external contributions**.
  We have taken great care in thoroughly commenting our code so that it is easy for new contributors to get up-to-speed and start contributing!
- The use of Go allows us to build an API to hook into the various parts of the fuzzer to build custom testing methodologies. See the [API Overview (WIP)](<https://github.com/crytic/medusa/wiki/API-Overview-(WIP)>) section in the Wiki for more details.
- Our forked version of go-ethereum, [`medusa-geth`](https://github.com/crytic/medusa-geth), exhibits behavior that is closer to that of the EVM in production environments.
- We can take the lessons we learned while developing Echidna to create a fuzzer that is just as feature-rich but with additional capabilities to
  create powerful and unique testing methodologies.

## Contributing

For information about how to contribute to this project, check out the [CONTRIBUTING](./CONTRIBUTING.md) guidelines.

## License

`medusa` is licensed and distributed under the [AGPLv3](./LICENSE).
