# `corpus clean`

The `corpus clean` command validates the call sequences stored in your corpus and removes any sequences that are no
longer executable:

```shell
medusa corpus clean [flags]
```

This is useful after contract refactors or ABI changes, when older corpus entries may no longer match the current
project state.

`medusa` reads your [project configuration](../project_configuration/overview.md), initializes the fuzzer, deploys the
target contracts, and then replays each saved sequence against a test chain. Any sequence that fails validation is
removed from disk.

## Requirements

- [`fuzzing.corpusDirectory`](../project_configuration/fuzzing_config.md#corpusdirectory) must be configured in your project config.
- If `--config` is not provided, `medusa` looks for `medusa.json` in the current working directory.

## Examples

```shell
# Clean the corpus configured in ./medusa.json
medusa corpus clean

# Clean the corpus for a specific project config
medusa corpus clean --config ./configs/medusa.json
```

## Supported Flags

### `--config`

The `--config` flag allows you to specify the path for your [project configuration](../project_configuration/overview.md)
file. If the `--config` flag is not used, `medusa` will look for a [`medusa.json`](../static/medusa.json) file in the
current working directory.

```shell
# Set config file path
medusa corpus clean --config myConfig.json
```
