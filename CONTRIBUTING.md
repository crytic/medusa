# Contribution Guidelines

## New Contributors

For more context about the project, check out the [README](./README.md).

To better understand how to make responsible open source contributions, consider the following guidance:

- [Finding ways to contribute to open source on GitHub](https://docs.github.com/en/get-started/exploring-projects-on-github/finding-ways-to-contribute-to-open-source-on-github)
- [Quickstart for GitHub Issues](https://docs.github.com/en/issues/tracking-your-work-with-issues/quickstart)
- [GitHub flow](https://docs.github.com/en/get-started/quickstart/github-flow)
- [Collaborating with pull requests](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests)

## Contributor & Reviewer Requirements

### Overview

When introducing changes to the project, note the following requirements:

- All changes to the main branch should be introduced via pull requests.
- All branches created for pull requests should follow the `dev/*` naming convention, e.g. `dev/coverage-reports`.
- Every pull request **must** be reviewed by at least one other peer prior to being merged into the main branch.
- Code **must** be supported on Linux, macOS, and Windows.
- Code **must** be sufficiently commented:
  - Every type, function, const, and other variables should be accompanied by [doc comments](https://tip.golang.org/doc/comment).
  - Inline comments should be provided for every block of code. These should explain what the block of code is aiming to achieve in plain english.
  - Inline comments should specify any non-trivial caveats with a given piece of code, considerations to make when maintaining it, etc.
  - Any considerations regarding potential weaknesses or future improvements to be made within your code should be accompanied by an inline comment prefixed with `// TODO: `
  - Comments should provide some value to a new contributor and improve code readability.
- Your changes **must not** contain copyrighted or licensed works without ensuring copyright law or licenses have not been violated.
- **Code must be compliant with the considerations laid out in the subsections below**.

If any of these requirements are violated, you should expect your pull request to be denied merging until appropriately remediated.

Pull request reviewers have a responsibility to uphold these standards. Even if a pull request is compliant with these requirements, a reviewer which identifies an opportunity to document some caveat (such as a `// TODO: ` comment) should request it be added prior to pull request approval.

### Linters

Several linters and security checkers are run on the PRs.

#### Go

To install

- `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`

To run

- `go fmt ./...`
- `golangci-lint run --timeout 5m0s`

#### Markdown/Json/Yaml

To install

- `npm install -g prettier`
- `npm install -g markdown-link-check@3.10.3`

To run

- `prettier '**.json' '**/*.md' '**/*.yml' '!(pkg)'`
- `find . -name '*.md' -print0 | xargs -0 -n1 markdown-link-check --config .github/workflows/resources/markdown_link_check.json`

To format (overwrite files)

- `prettier '**.json' '**/*.md' '**/*.yml' '!(pkg)' -w`

#### Github action

To install

- `go install github.com/rhysd/actionlint/cmd/actionlint@latest`

To run

- `actionlint`

### Cross-platform considerations

- Ensure file/directory names do not exceed 32 characters in length to minimize filepath length issues on Windows. File/directory names should be shorter than this where possible.
- Ensure your code supports LF and CRLF line endings. While Linux/macOS us LF line endings (`\n`), Windows uses CRLF line endings (`\r\n`). This is important to support when processing text files (e.g., performing source mapping operations).
- Ensure for file operations, you use the `filepath` package rather than `path` where possible. `filepath` respects system path separators, while `path` does not.
  - Windows file paths use backslashes (`\\`) while other operating systems tend to use forward-slashes (`/`). Many APIs will support forward slashes on Windows, but some may not. This is generally important to note when constructing file paths or performing equality checks.
  - `path` uses forward-slashes to construct consistent path strings across systems. `filepath` will use the system-default.

### Serialization considerations

- Ensure JSON keys are `camelCase` rather than `snake_case`, where possible.

### Nix considerations

- If any dependencies are added or removed, the `vendorHash` property in ./flake.nix will need to be updated. To do so, run `nix build`. If it works, you're good to go. If a change is required, you'll see an error that looks like the following. Replace the `specified` value of `vendorHash` in the medusa package of flake.nix with what nix actually `got`.

```
error: hash mismatch in fixed-output derivation '/nix/store/sfgmkr563pzyxzllpmwxdbdxgrav8y1p-medusa-0.1.8-go-modules.drv':
         specified: sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=
            got:    sha256-12Xkg5dzA83HQ2gMngXoLgu1c9KGSL6ly5Qz/o8U++8=
```

## License

The license for this software can be found [here](./LICENSE).
