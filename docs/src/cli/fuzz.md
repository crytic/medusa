# `fuzz`

The `fuzz` command will initiate a fuzzing campaign:

```shell
medusa fuzz [flags]
```

## Supported Flags

### `--config`

The `--config` flag allows you to specify the path for your [project configuration](../project_configuration/overview.md)
file. If the `--config` flag is not used, `medusa` will look for a [`medusa.json`](../static/medusa.json) file in the
current working directory.

```shell
# Set config file path
medusa fuzz --out myConfig.json
```

### `--compilation-target`

The `--compilation-target` flag allows you to specify the compilation target. If you are using `crytic-compile`, please review the
warning [here](../project_configuration/compilation_config.md#target) about changing the compilation target.

```shell
# Set compilation target
medusa fuzz --target TestMyContract.sol
```

### `--workers`

The `--workers` flag allows you to update the number of threads that will perform parallelized fuzzing (equivalent to
[`fuzzing.workers`](../project_configuration/fuzzing_config.md#workers))

```shell
# Set workers
medusa fuzz --workers 20
```

### `--timeout`

The `--timeout` flag allows you to update the duration of the fuzzing campaign (equivalent to
[`fuzzing.timeout`](../project_configuration/fuzzing_config.md#timeout))

```shell
# Set timeout
medusa fuzz --timeout 100
```

### `--test-limit`

The `--test-limit` flag allows you to update the number of transactions to run before stopping the fuzzing campaign
(equivalent to [`fuzzing.testLimit`](../project_configuration/fuzzing_config.md#testlimit))

```shell
# Set test limit
medusa fuzz --test-limit 100000
```

### `--seq-len`

The `--seq-len` flag allows you to update the length of a call sequence (equivalent to
[`fuzzing.callSequenceLength`](../project_configuration/fuzzing_config.md#callsequencelength))

```shell
# Set sequence length
medusa fuzz --seq-len 50
```

### `--target-contracts`

The `--target-contracts` flag allows you to update the target contracts for fuzzing (equivalent to
[`fuzzing.targetContracts`](../project_configuration/fuzzing_config.md#targetcontracts))

```shell
# Set target contracts
medusa fuzz --target-contracts "TestMyContract, TestMyOtherContract"
```

### `--corpus-dir`

The `--corpus-dir` flag allows you to set the path for the corpus directory (equivalent to
[`fuzzing.corpusDirectory`](../project_configuration/fuzzing_config.md#corpusdirectory))

```shell
# Set corpus directory
medusa fuzz --corpus-dir corpus
```

### `--senders`

The `--senders` flag allows you to update `medusa`'s senders (equivalent to
[`fuzzing.senderAddresses`](../project_configuration/fuzzing_config.md#senderaddresses))

```shell
# Set sender addresses
medusa fuzz --senders "0x50000,0x60000,0x70000"
```

### `--deployer`

The `--deployer` flag allows you to update `medusa`'s contract deployer (equivalent to
[`fuzzing.deployerAddress`](../project_configuration/fuzzing_config.md#deployeraddress))

```shell
# Set deployer address
medusa fuzz --deployer "0x40000"
```

### `--use-slither`

The `--use-slither` flag allows you to run Slither on the codebase to extract valuable constants for mutation testing.
Equivalent to [`slither.useSlither`](../project_configuration/slither_config.md#useslither). Note
that if there are cached results (via [`slither.CachePath`](../project_configuration/slither_config.md#cachepath)) then
the cache will be used.

```shell
# Run slither and attempt to use cache, if available
medusa fuzz --use-slither
```

### `--use-slither-force`

The `--use-slither-force` flag is similar to `--use-slither` except the cache at `slither.CachePath` will be
overwritten.

```shell
# Run slither and overwrite the cache
medusa fuzz --use-slither-force
```

### `--fail-fast`

The `--fail-fast` flag enables fast failure (equivalent to
[`testing.StopOnFailedTest`](../project_configuration/testing_config.md#stoponfailedtest))

```shell
# Enable fast failure
medusa fuzz --fail-fast
```

### `--trace-all`

The `--trace-all` flag allows you to retrieve an execution trace for each element of a call sequence that triggered a test
failure (equivalent to
[`testing.traceAll`](../project_configuration/testing_config.md#traceall)

```shell
# Trace each call
medusa fuzz --trace-all
```

### `--no-color`

The `--no-color` flag disables colored console output (equivalent to
[`logging.NoColor`](../project_configuration/logging_config.md#nocolor))

```shell
# Disable colored output
medusa fuzz --no-color
```

### `--explore`

The `--explore` flag enables exploration mode. This sets the [`StopOnFailedTest`](../project_configuration/testing_config.md#stoponfailedtest) and [`StopOnNoTests`](../project_configuration/testing_config.md#stoponnotests)
fields to `false` and turns off assertion, property, and optimization testing.

```shell
# Enable exploration mode
medusa fuzz --explore
```
