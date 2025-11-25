# Logging Configuration

The logging configuration defines the parameters for logging to console and/or file.

### `level`

- **Type**: String
- **Description**: The log level will determine which logs are emitted or discarded. If `level` is "info" then all logs
  with informational level or higher will be logged. The supported values for `level` are "trace", "debug", "info", "warn", "error",
  and "panic".
- **Default**: "info"

### `logDirectory`

- **Type**: String
- **Description**: Describes what directory log files should be output to. Having a non-empty `logDirectory` value will
  enable "file logging" which will result in logs to be output to both console and file. Note that the directory path is
  _relative_ to the directory containing the project configuration file.
- **Default**: ""

### `noColor`

- **Type**: Boolean
- **Description**: Disables colored output to console.
- **Default**: `false`

### `enableTUI`

- **Type**: Boolean
- **Description**: Whether to enable the Terminal User Interface (TUI) for an interactive fuzzing experience. The TUI provides real-time visualization of fuzzing progress, worker status, test results, and interactive trace viewing. See the [--tui flag documentation](../cli/fuzz.md#--tui) for keyboard controls and features.
- **Default**: `false`
