# First Steps

After installation, you are ready to use `medusa` on your first codebase. This chapter will walk you through initializing
`medusa` for a project and then starting to fuzz.

To initialize medusa for a project, `cd` into your project and run [`medusa init`](../cli/init.md):

```shell
# Change working directory
cd my_project

# Initialize medusa
medusa init
```

This will create a `medusa.json` file which holds a large number of [configuration options](../project_configuration/overview.md).
`medusa` will use this configuration file to determine how and what to fuzz.

All there is left to do now is to run `medusa` on some fuzz tests:

```shell
medusa fuzz --target-contracts "TestContract" --test-limit 10_000
```

The `--target-contracts` flag tells `medusa` which contracts to run fuzz tests on. You can specify more than one
contract to fuzz test at once (e.g. `--target-contracts "TestContract, TestOtherContract"`). The `--test-limit` flag
tells `medusa` to execute `10_000` transactions before stopping the fuzzing campaign.

> Note: The target contracts and the test limit can also be configured via the project configuration file, which is the
> **recommended** route. The `--target-contracts` flag is equivalent to the
> [`fuzzing.targetContracts`](../project_configuration/fuzzing_config.md#targetcontracts) configuration option and the
> `-test-limit` flag is equivalent to the [`fuzzing.testLimit`](../project_configuration/fuzzing_config.md#testlimit)
> configuration option.

It is recommended to review the [Configuration Overview](../project_configuration/overview.md) next and learn more about
[`medusa`'s CLI](../cli/overview.md).
