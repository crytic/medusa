## General Overview of Medusa

### Introduction

Medusa is a cross-platform smart contract fuzzer inspired by Echidna. It provides parallelized fuzz testing of smart contracts through CLI or its Go API, allowing for custom user-extended testing methodologies.

### Features

- **Parallel fuzzing and testing:** Medusa supports parallel fuzzing and testing methodologies across multiple workers (threads).
- **Assertion and property testing:** It provides built-in support for writing basic Solidity property tests and assertion tests.
- **Mutational value generation:** Medusa generates mutational values based on compilation and runtime values.
- **Coverage collecting:** Coverage increasing call sequences are stored in the corpus.
- **Coverage guided fuzzing:** Coverage increasing call sequences from the corpus are mutated to further guide the fuzzing campaign.
- **Extensible low-level testing API:** Medusa offers an extensible low-level testing API through events and hooks provided throughout the fuzzer, workers, and test chains.

### Usage

Medusa can be run through the CLI or a configuration file driven format.

**CLI:**

```
medusa fuzz --target contract.sol --deployment-order ContractName
```

**Configuration file driven:**

1. Create a `medusa.json` file in the project directory.
2. Set the `"target"` field to point to the file/directory for contract compilation.
3. Specify the names of contracts to be deployed and tested in the `"targetContracts"` field.
4. Run `medusa fuzz`.

### Benefits of Medusa

- **Lower barrier of entry:** Medusa is written in Go, making it easier for external contributions.
- **Customizable testing methodologies:** The Go API allows users to hook into the fuzzer and create custom testing methodologies.
- **Closer to production EVM behavior:** Medusa uses a forked version of go-ethereum (`medusa-geth`) that exhibits behavior closer to that of the EVM in production environments.
- **Leverages lessons from Echidna:** Medusa builds upon the lessons learned from developing Echidna to create a feature-rich fuzzer with unique testing capabilities.
