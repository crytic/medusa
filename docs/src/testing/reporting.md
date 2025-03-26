# Medusa Reports

Medusa provides two types of reports to help you analyze the results of your fuzzing campaigns:

1. **Coverage Reports**: Show which parts of your contracts were executed during fuzzing
2. **Revert Reports**: Help you understand which functions are reverting and why (experimental)

## Coverage Reports

Coverage reports help you understand which parts of your contract code have been executed during fuzzing. This is valuable for:

- Identifying code paths that haven't been tested
- Focusing your fuzzing efforts on uncovered areas
- Measuring the effectiveness of your fuzzing campaign

Medusa supports two types of coverage report formats:

### HTML Coverage Reports

HTML reports provide a visual representation of code coverage with highlighted source code. This is the most user-friendly format for quickly understanding coverage.

### LCOV Coverage Reports

LCOV reports are machine-readable and useful for integrating with other tools or continuous integration systems.

### Enabling Coverage Reports

Enable coverage reporting by setting the `corpusDirectory` and `coverageReports` options in your configuration file:

```json
{
  "corpusDirectory": "corpus",
  "coverageReports": ["lcov", "html"]
}
```

If a `corpusDirectory` is not provided, the report(s) will be saved at `crytic-export/coverage`.
### Viewing HTML Coverage Reports

The HTML report is automatically generated at `corpus/coverage/coverage_report.html`. Open this file in any web browser to view your coverage.

### Using LCOV Reports

LCOV files can be used with various tools:

#### Generate HTML from LCOV

First, install the LCOV tools:

Linux:
```bash
apt-get install lcov
```

MacOS:
```bash
brew install lcov
```

Then generate HTML from the LCOV data:
```bash
genhtml corpus/coverage/lcov.info --output-dir corpus --rc derive_function_end_line=0
```

> 🚩WARNING
> The `derive_function_end_line` flag is required to prevent the `genhtml` tool from crashing when processing Solidity source code.

Open the `corpus/index.html` file in your browser to view the report.

#### View Coverage in VSCode

1. Install the [Coverage Gutters](https://marketplace.visualstudio.com/items?itemName=ryanluker.vscode-coverage-gutters) extension
2. Right-click in a project file and select `Coverage Gutters: Display Coverage`

## Revert Reports

Revert reports are an **experimental feature** that helps you understand which functions in your contract frequently revert during fuzzing and why. This is particularly useful for:

- Debugging your fuzzing harness
- Identifying input validation issues
- Understanding constraints that are limiting your fuzzer's exploration
- Improving your test cases and assertions

### Enabling Revert Reports

To enable revert reports, you need to set the `revertReporterEnabled` parameter to `true` in your configuration file:

```json
{
  "fuzzing": {
    "revertReporterEnabled": true,
    "corpusDirectory": "corpus"
  }
}
```

If a `corpusDirectory` is not provided, the reports will be saved at `crytic-export/coverage`.

### Viewing Revert Reports

Two report files are generated:

1. `corpus/coverage/revert_report.html` - HTML visualization of revert metrics
2. `corpus/coverage/revert_report.json` - Machine-readable JSON format

The HTML report provides detailed statistics on:
- Which functions revert most frequently
- Common revert reasons for each function
- Comparison with previous fuzzing runs (when available)

### Benefits of Revert Reports

Revert reports are especially helpful during the early stages of fuzzing development:

- **Harness Development**: Identify issues in your fuzzing setup that prevent effective testing
- **Input Improvement**: Determine which inputs cause frequent reverts and refine your value generation
- **Contract Constraints**: Understand logical constraints in your contract that may be limiting fuzzing effectiveness