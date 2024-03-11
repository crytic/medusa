# Testing Overview
Medusa is a smart contract fuzzer that supports three main testing modes:

- **Property testing:** Checks if certain properties of the contract hold true under all possible inputs.
- **Assertion testing:** Verifies that specific assertions within the contract hold true during execution.
- **Optimization testing:** Attempts to maximize the return value of a given function.

## Writing Tests

- **Property tests:** Functions prefixed with a specified prefix (e.g., `fuzz_`) that take no arguments and return a boolean indicating success.
- **Assertion tests:** Functions that contain `assert` statements or cause the EVM to panic.
- **Optimization tests:** Functions prefixed with a specified prefix (e.g., `optimize_`) that take no arguments and return an integer.

## Running Tests

Medusa can be run with one or more testing modes enabled via command-line flags or a configuration file.

## Results

Medusa reports the results of its testing campaign, including:

- Failed property tests and the call sequences that caused them.
- Failed assertions and the call sequences that caused them.
- The maximum return value achieved in optimization tests and the call sequence that produced it.

## Benefits

- Comprehensive testing through multiple modes.
- Parallel execution for faster results.
- Customizable testing framework.
- User-friendly CLI and configuration file format.
