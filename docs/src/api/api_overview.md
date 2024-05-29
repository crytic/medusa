# API Overview (WIP)

`medusa` offers a lower level API to hook into various parts of the fuzzer, its workers, and underlying chains. Although assertion and property testing are two built-in testing providers, they are implementing using events and hooks offered throughout the `Fuzzer`, `FuzzerWorker`(s), and underlying `TestChain`. These same hooks can be used by external developers wishing to implement their own customing testing methodology. In the sections below, we explore some of the relevant components throughout `medusa`, their events/hooks, an example of creating custom testing methodology with it.

## Component overview

A rudimentary description of the objects/providers and their roles are explained below.

### Data types

- `ProjectConfig`: This defines the configuration for the Fuzzer, including the targets to compile, deploy, and how to fuzz or test them.

- `ValueSet`: This is an object that acts as a dictionary of values, used in mutation operations. It is populated at compilation time with some rudimentary static analysis.

- `Contract`: Can be thought of as a "contract definition", it is a data type which stores the name of the contract, and a reference to the underlying `CompiledContract`, a definition derived from compilation, containing the bytecode, source maps, ABI, etc.

- `CallSequence`: This represents a list of `CallSequenceElement`s, which define a transaction to send, the suggested block number and timestamp delay to use, and stores a reference to the block/transaction/results when it is executed (for later querying in tests). They are used to generate and execute transaction sequences in the fuzzer.

- `CoverageMaps` define a list of `CoverageMap` objects, which record all instruction offsets executed for a given contract address and code hash.

- `TestCase` defines the interface for a test that the `Fuzzer` will track. It simply defines a name, ID, status (not started, running, passed, failed) and message for the `Fuzzer`.

### Providers

- `ValueGenerator`: This is an object that provides methods to generate values of different kinds for transactions. Examples include the `RandomValueGenerator` and superceding `MutationalValueGenerator`. They are provided a `ValueSet` by their worker, which they may use in generation operations.

- `TestChain`: This is a fake chain that operates on fake block structures created for the purpose of testing. Rather than operating on `types.Transaction` (which requires signing), it operates on `core.Message`s, which are derived from transactions and simply allow you to set the `sender` field. It is responsible for:

  - Maintaining state of the chain (blocks, transactions in them, results/receipts)
  - Providing methods to create blocks, add transactions to them, commit them to chain, revert to previous block numbers.
  - Allowing spoofing of block number and timestamp (commiting block number 1, then 50, jumping 49 blocks ahead), while simulating the existence of intermediate blocks.
  - Provides methods to add tracers such as `evm.Logger` (standard go-ethereum tracers) or extend them with an additional interface (`TestChainTracer`) to also store any captured traced information in the execution results. This allows you to trace EVM execution for certain conditions, store results, and query them at a later time for testing.

- `Fuzzer`: This is the main provider for the fuzzing process. It takes a `ProjectConfig` and is responsible for:

  - Housing data shared between the `FuzzerWorker`s such as contract definitions, a `ValueSet` derived from compilation to use in value generation, the reference to `Corpus`, the `CoverageMaps` representing all coverage achieved, as well as maintaining `TestCase`s registered to it and printing their results.
  - Compiling the targets defined by the project config and setting up state.
  - Provides methods to start/stop the fuzzing process, add additional compilation targets, access the initial value set prior to fuzzing start, access corpus, config, register new test cases and report them finished.
  - Starts the fuzzing process by creating a "base" `TestChain`, deploys compiled contracts, replays all corpus sequences to measure existing coverage from previous fuzzing campaign, then spawns as many `FuzzerWorker`s as configured on their own goroutines ("threads") and passes them the "base" `TestChain` (which they clone) to begin the fuzzing operation.
    - Respawns `FuzzerWorker`s when they hit a config-defined reset limit for the amount of transaction sequences they should process before destroying themselves and freeing memory.
    - Maintains the context for when fuzzing should stop, which all workers track.

- `FuzzerWorker`: This describes an object spawned by the `Fuzzer` with a given "base" `TestChain` with target contracts already deployed, ready to be fuzzed. It clones this chain, then is called upon to begin creating fuzz transactions. It is responsible for:
  - Maintaining a reference to the parent `Fuzzer` for any shared information between it and other workers (`Corpus`, total `CoverageMaps`, contract definitions to match deployment's bytecode, etc)
  - Maintaining its own `TestChain` to run fuzzed transaction sequences.
  - Maintaining its own `ValueSet` which derives from the `Fuzzer`'s `ValueSet` (populated by compilation or user-provided values through API), as each `FuzzerWorker` may populate its `ValueSet` with different runtime values depending on their own chain state.
  - Spawning a `ValueGenerator` which uses the `ValueSet`, to generate values used to construct fuzzed transaction sequences.
  - Most importantly, it continuously:
    - Generates `CallSequence`s (a series of transactions), plays them on its `TestChain`, records the results of in each `CallSequenceElement`, and calls abstract/hookable "test functions" to indicate they should perform post-tx tests (for which they can return requests for a shrunk test sequence).
    - Updates the total `CoverageMaps` and `Corpus` with the current `CallSequence` if the most recent call increased coverage.
    - Processes any shrink requests from the previous step (shrink requests can define arbitrary criteria for shrinking).
  - Eventually, hits the config-defined reset limit for how many sequences it should process, and destroys itself to free all memory, expecting the `Fuzzer` to respawn another in its place.

## Creating a project configuration

`medusa` is config-driven. To begin a fuzzing campaign on an API level, you must first define a project configuration so the fuzzer knows what contracts to compile, deploy, and how it should operate.

When using `medusa` over command-line, it operates a project config similarly (see [docs](https://github.com/trailofbits/medusa/wiki/Project-Configuration) or [example](https://github.com/trailofbits/medusa/wiki/Example-Project-Configuration-File)). Similarly, interfacing with a `Fuzzer` requires a `ProjectConfig` object. After importing `medusa` into your Go project, you can create one like this:

```go
// Initialize a default project config with using crytic-compile as a compilation platform, and set the target it should compile.
projectConfig := config.GetDefaultProjectConfig("crytic-compile")
err := projectConfig.Compilation.SetTarget("contract.sol")
if err != nil {
    return err
}

// You can edit any of the values as you please.
projectConfig.Fuzzing.Workers = 20
projectConfig.Fuzzing.DeploymentOrder = []string{"TestContract1", "TestContract2"}
```

You may also instantiate the whole config in-line with all the fields you'd like, setting the underlying platform config yourself.

> **NOTE**: The `CompilationConfig` and `PlatformConfig` WILL BE deprecated and replaced with something more intuitive in the future, as the `compilation` package has not been updated since the project's inception, prior to the release of generics in go 1.18.

## Creating and starting the fuzzer

After you have created a `ProjectConfig`, you can create a new `Fuzzer` with it, and tell it to start:

```go
	// Create our fuzzer
	fuzzer, err := fuzzing.NewFuzzer(*projectConfig)
	if err != nil {
		return err
	}

	// Start the fuzzer
	err = fuzzer.Start()
	if err != nil {
		return err
	}

    // Fetch test cases results
    testCases := fuzzer.TestCases()
[...]
```

> **Note**: `Fuzzer.Start()` is a blocking operation. If you wish to stop, you must define a TestLimit or Timeout in your config. Otherwise start it on another goroutine and call `Fuzzer.Stop()` to stop it.

## Events/Hooks

### Events

Now it may be the case that you wish to hook the `Fuzzer`, `FuzzerWorker`, or `TestChain` to provide your own functionality. You can add your own testing methodology, and even power it with your own low-level EVM execution tracers to store and query results about each call.

There are a few events/hooks that may be useful of the bat:

The `Fuzzer` maintains event emitters for the following events under `Fuzzer.Events.*`:

- `FuzzerStartingEvent`: Indicates a `Fuzzer` is starting and provides a reference to it.

- `FuzzerStoppingEvent`: Indicates a `Fuzzer` has just stopped all workers and is about to print results and exit.

- `FuzzerWorkerCreatedEvent`: Indicates a `FuzzerWorker` was created by a `Fuzzer`. It provides a reference to the `FuzzerWorker` spawned. The parent `Fuzzer` can be accessed through `FuzzerWorker.Fuzzer()`.
- `FuzzerWorkerDestroyedEvent`: Indicates a `FuzzerWorker` was destroyed. This can happen either due to hitting the config-defined worker reset limit or the fuzzing operation stopping. It provides a reference to the destroyed worker (for reference, though this should not be stored, to allow memory to free).

The `FuzzerWorker` maintains event emiters for the following events under `FuzzerWorker.Events.*`:

- `FuzzerWorkerChainCreatedEvent`: This indicates the `FuzzerWorker` is about to begin working and has created its chain (but not yet copied data from the "base" `TestChain` the `Fuzzer` provided). This offers an opportunity to attach tracers for calls made during chain setup. It provides a reference to the `FuzzerWorker` and its underlying `TestChain`.

- `FuzzerWorkerChainSetupEvent`: This indicates the `FuzzerWorker` is about to begin working and has both created its chain, and copied data from the "base" `TestChain`, so the initial deployment of contracts is complete and fuzzing is ready to begin. It provides a reference to the `FuzzerWorker` and its underlying `TestChain`.

- `CallSequenceTesting`: This indicates a new `CallSequence` is about to be generated and tested by the `FuzzerWorker`. It provides a reference to the `FuzzerWorker`.

- `CallSequenceTested`: This indicates a `CallSequence` was just tested by the `FuzzerWorker`. It provides a reference to the `FuzzerWorker`.

- `FuzzerWorkerContractAddedEvent`: This indicates a contract was added on the `FuzzerWorker`'s underlying `TestChain`. This event is emitted when the contract byte code is resolved to a `Contract` definition known by the `Fuzzer`. It may be emitted due to a contract deployment, or the reverting of a block which caused a SELFDESTRUCT. It provides a reference to the `FuzzerWorker`, the deployed contract address, and the `Contract` definition that it was matched to.

- `FuzzerWorkerContractDeletedEvent`: This indicates a contract was removed on the `FuzzerWorker`'s underlying `TestChain`. It may be emitted due to a contract deployment which was reverted, or a SELFDESTRUCT operation. It provides a reference to the `FuzzerWorker`, the deployed contract address, and the `Contract` definition that it was matched to.

The `TestChain` maintains event emitters for the following events under `TestChain.Events.*`:

- `PendingBlockCreatedEvent`: This indicates a new block is being created but has not yet been committed to the chain. The block is empty at this point but will likely be populated. It provides a reference to the `Block` and `TestChain`.

- `PendingBlockAddedTxEvent`: This indicates a pending block which has not yet been commited to chain has added a transaction to it, as it is being constructed. It provides a reference to the `Block`, `TestChain`, and index of the transaction in the `Block`.

- `PendingBlockCommittedEvent`: This indicates a pending block was committed to chain as the new head. It provides a reference to the `Block` and `TestChain`.

- `PendingBlockDiscardedEvent`: This indicates a pending block was not committed to chain and was instead discarded.

- `BlocksRemovedEvent`: This indicates blocks were removed from the chain. This happens when a chain revert to a previous block number is invoked. It provides a reference to the `Block` and `TestChain`.

- `ContractDeploymentsAddedEvent`: This indicates a new contract deployment was detected on chain. It provides a reference to the `TestChain`, as well as information captured about the bytecode. This may be triggered on contract deployment, or the reverting of a SELFDESTRUCT operation.

- `ContractDeploymentsRemovedEvent`: This indicates a previously deployed contract deployment was removed from chain. It provides a reference to the `TestChain`, as well as information captured about the bytecode. This may be triggered on revert of a contract deployment, or a SELFDESTRUCT operation.

### Hooks

The `Fuzzer` maintains hooks for some of its functionality under `Fuzzer.Hooks.*`:

- `NewValueGeneratorFunc`: This method is used to create a `ValueGenerator` for each `FuzzerWorker`. By default, this uses a `MutationalValueGenerator` constructed with the provided `ValueSet`. It can be replaced to provide a custom `ValueGenerator`.

- `TestChainSetupFunc`: This method is used to set up a chain's initial state before fuzzing. By default, this method deploys all contracts compiled and marked for deployment in the `ProjectConfig` provided to the `Fuzzer`. It only deploys contracts if they have no constructor arguments. This can be replaced with your own method to do custom deployments.

  - **Note**: We do not recommend replacing this for now, as the `Contract` definitions may not be known to the `Fuzzer`. Additionally, `SenderAddresses` and `DeployerAddress` are the only addresses funded at genesis. This will be updated at a later time.

- `CallSequenceTestFuncs`: This is a list of functions which are called after each `FuzzerWorker` executed another call in its current `CallSequence`. It takes the `FuzzerWorker` and `CallSequence` as input, and is expected to return a list of `ShinkRequest`s if some interesting result was found and we wish for the `FuzzerWorker` to shrink the sequence. You can add a function here as part of custom post-call testing methodology to check if some property was violated, then request a shrunken sequence for it with arbitrary criteria to verify the shrunk sequence satisfies your requirements (e.g. violating the same property again).

### Extending testing methodology

Although we will build out guidance on how you can solve different challenges or employ different tests with this lower level API, we intend to wrap some of this into a higher level API that allows testing complex post-call/event conditions with just a few lines of code externally. The lower level API will serve for more granular control across the system, and fine tuned optimizations.

To ensure testing methodology was agnostic and extensible in `medusa`, we note that both assertion and property testing is implemented through the abovementioned events and hooks. When a higher level API is introduced, we intend to migrate these test case providers to that API.

For now, the built-in `AssertionTestCaseProvider` (found [here](https://github.com/trailofbits/medusa/blob/8036697794481b7bf9fa78c922ec7fa6a8a3005c/fuzzing/test_case_assertion_provider.go)) and its test cases (found [here](https://github.com/trailofbits/medusa/blob/8036697794481b7bf9fa78c922ec7fa6a8a3005c/fuzzing/test_case_assertion.go)) are an example of code that _could_ exist externally outside of `medusa`, but plug into it to offer extended testing methodology. Although it makes use of some private variables, they can be replaced with public getter functions that are available. As such, if assertion testing didn't exist in `medusa` natively, you could've implemented it yourself externally!

In the end, using it would look something like this:

```go
	// Create our fuzzer
	fuzzer, err := fuzzing.NewFuzzer(*projectConfig)
	if err != nil {
		return err
	}

	// Attach our custom test case provider
	attachAssertionTestCaseProvider(fuzzer)

	// Start the fuzzer
	err = fuzzer.Start()
	if err != nil {
		return err
	}
```
