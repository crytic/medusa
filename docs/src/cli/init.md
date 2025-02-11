# `init`

The `init` command will generate the project configuration file within your current working directory:

```shell
medusa init [platform] [flags]
```

By default, the project configuration file will be named `medusa.json`. You can learn more about `medusa`'s project
configuration [here](../project_configuration/overview.md) and also view an [example project configuration file](../static/medusa.json).

Invoking this command without a `platform` argument will result in `medusa` using `crytic-compile` as the default compilation platform.
Currently, the only other supported platform is `solc`. If you are using a compilation platform such as Foundry or Hardhat,
it is best to use `crytic-compile`.

## Supported Flags

### `--out`

The `--out` flag allows you to specify the output path for the project configuration file. Thus, you can name the file
something different from `medusa.json` or have the configuration file be placed elsewhere in your filesystem.

```shell
# Set config file path
medusa init --out myConfig.json
```

### `--compilation-target`

The `--compilation-target` flag allows you to specify the compilation target. If you are using `crytic-compile`, please review the
warning [here](../project_configuration/compilation_config.md#target) about changing the compilation target.

```shell
# Set compilation target
medusa init --compilation-target TestMyContract.sol
```
