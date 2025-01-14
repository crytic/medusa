# Slither Configuration

The [Slither](https://github.com/crytic/slither) configuration defines the parameters for using Slither in `medusa`.
Currently, we use Slither to extract interesting constants from the target system. These constants are then used in the
fuzzing process to try to increase coverage. Note that if Slither fails to run for some reason, we will still try our
best to mine constants from each contract's AST so don't worry!

- > 🚩 We _highly_ recommend using Slither and caching the results. Basically, don't change this configuration unless
  > absolutely necessary. The constants identified by Slither are shown to greatly improve system coverage and caching
  > the results will improve the speed of medusa.

### `useSlither`

- **Type**: Boolean
- **Description**: If `true`, Slither will be run on the target system and useful constants will be extracted for fuzzing.
  If `cachePath` is a non-empty string (which it is by default), then `medusa` will first check the cache before running
  Slither.
- **Default**: `true`

### `cachePath`

- **Type**: String
- **Description**: If `cachePath` is non-empty, Slither's results will be cached on disk. When `medusa` is re-run, these
  cached results will be used. We do this for performance reasons since re-running Slither each time `medusa` is restarted
  is computationally intensive for complex projects. We recommend disabling caching (by making `cachePath` an empty string)
  if the target codebase changes. If the code remains constant during the fuzzing campaign, we recommend to use the cache.
- **Default**: `slither_results.json`
