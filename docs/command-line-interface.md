`medusa` can run parallelized fuzz testing of smart contracts through its Command Line Interface (CLI). The CLI supports three main commands and each command has a variety of flags:

1. `medusa init [platform]` will initialize `medusa`'s project configuration file
2. `medusa fuzz` will begin the fuzzing campaign.
3. `medusa completion <shell>` will provide an autocompletion script for a given shell

Let's look at each command.

> **Note**: We highly recommend reading more about `medusa`'s [project configuration](./Project-Configuration.md) parameters before diving into `medusa`'s CLI capabilities.

## Initializing a project configuration

To create a `medusa` project configuration, invoke `medusa init [platform]` to create a [configuration file](./Example-Project-Configuration-File.md) for the provided platform within your current working directory. Invoking this command without a `platform` argument will result in `medusa` using `crytic-compile` as the default compilation platform.

> Note that the output of `medusa init`, which is equivalent to `medusa init crytic-compile`, is considered the **default configuration** of `medusa`

While running `medusa init`, you have access to two flags:

1. `medusa init [platform] --out myConfig.json`: The `--out` value will determine the output path where your project configuration file will be outputted. Without `--out`, the default output path is `medusa.json` in the current working directory.
2. `medusa init [platform] --target myContract.sol`: The `--target` value will determine the compilation target.

## Running a fuzzing campaign

After you have a project configuration setup, you can now run a fuzzing campaign.

To run a fuzzing campaign, invoke `medusa fuzz`. The `fuzz` command supports a variety of flags:

- `medusa fuzz --config myConfig.json`: Will use the configuration in `myConfig.json` as the project configuration. If `--config` is not set, `medusa` will look for a `medusa.json` file in the current working directory
- `medusa fuzz --target myContract.sol`: Will set the compilation target to `myContract.sol`
- `medusa fuzz --workers 20`: Will set the number of `workers` to 20 threads
- `medusa fuzz --timeout 1000`: Will set the `timeout` to 1000 seconds
- `medusa fuzz --test-limit 50000`: Will set the `testLimit` to 50000 function calls
- `medusa fuzz --seq-len 50`: Will set the `callSequenceLength` to 50 transactions
- `medusa fuzz --deployment-order "FirstContract,SecondContract"`: Will set the deployment order to `[FirstContract, SecondContract]`
- `medusa fuzz --corpus-dir myCorpus`: Will set the corpus directory _path_ to `myCorpus`
- `medusa fuzz --senders "0x10000,0x20000,0x30000"`: Will set the `senderAdddresses` to `[0x10000, 0x20000, 0x30000]`
- `medusa fuzz --deployer "0x10000"`: Will set the `deployerAddress` to `0x10000`
- `medusa fuzz --assertion-mode`: Will set `assertionTesting.enabled` to `true`
- `medusa fuzz --optimization-mode`: Will set `optimizationTesting.enabled` to `true`
- `medusa fuzz --trace-all`: Will set `traceAll` to `true`

Note that the `fuzz` command will use both the project configuration file in addition to any flags to determine the _final_ project configuration. Thus, it uses both of them in tandem.

This results in four different ways to run `medusa`:

1. `medusa fuzz`: Run `medusa` using the configuration in `medusa.json` with no CLI updates.
   > 🚩 If `medusa.json` is not found, we will use the **default configuration**.
2. `medusa fuzz --workers 20 --test-limit 50000`: Run `medusa` using the configuration in `medusa.json` and override the `workers` and `testLimit` parameters.
   > 🚩 If `medusa.json` is not found, we will use the **default configuration** and override the `workers` and `testLimit` parameters.
3. `medusa fuzz --config myConfig.json`: Run `medusa` using the configuration in `myConfig.json` with no CLI updates.
   > 🚩 If `myConfig.json` is not found, `medusa` will throw an error
4. `medusa fuzz --config myConfig.json --workers 20 --test-limit 50000`: Run `medusa` using the configuration in `myConfig.json` and override the `workers` and `testLimit` parameters.
   > 🚩 If `myConfig.json` is not found, `medusa` will throw an error

## Autocompletion

`medusa` also provides the ability to generate autocompletion scripts for a given shell. Once the autocompletion script is ran for a given shell, `medusa`'s commands and flags can now be tab-autocompleted. The following shells are supported:

1. bash
2. zsh
3. Powershell

To understand how to run the autocompletion script for a given shell, run the following command

```bash
medusa completion --help
```

Once you know how to run the autocompletion script, retrieve the script for that given shell using the following command.

```bash
medusa completion <shell>
```
