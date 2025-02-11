## Coverage Reports

### Generating HTML Report from LCOV

Enable coverage reporting by setting the `corpusDirectory` key in the configuration file and setting the `coverageReports` key to `["lcov", "html"]`.

```json
{
  "corpusDirectory": "corpus",
  "coverageReports": ["lcov", "html"]
}
```

### Install lcov and genhtml

Linux:

```bash
apt-get install lcov
```

MacOS:

```bash
brew install lcov
```

### Generate LCOV Report

```bash
genhtml corpus/coverage/lcov.info --output-dir corpus --rc derive_function_end_line=0
```

> [!WARNING]  
> ** The `derive_function_end_line` flag is required to prevent the `genhtml` tool from crashing when processing the Solidity source code. **

Open the `corpus/index.html` file in your browser or follow the steps to use VSCode below.

### View Coverage Report in VSCode with Coverage Gutters

Install the [Coverage Gutters](https://marketplace.visualstudio.com/items?itemName=ryanluker.vscode-coverage-gutters) extension.

Then, right click in a project file and select `Coverage Gutters: Display Coverage`.
