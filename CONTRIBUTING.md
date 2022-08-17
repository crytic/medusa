# Contribution Guidelines

## New Contributors
For more context about the project, check out the [README](./README.md).

To better understand how to make responsible open source contributions, consider the following guidance:
- TODO1
- TODO2

## Contributor & Reviewer Requirements
When introducing changes to the project, note the following requirements:
- All changes to the main branch should be introduced via pull requests.
- Every pull request should be reviewed by at least one other peer prior to being merged into the main branch.
- Code MUST be supported on Linux, macOS, and Windows.
- Code MUST be sufficiently commented:
  - Every type, function, const, and other variables should be accompanied by [doc comments](https://tip.golang.org/doc/comment).
  - Inline comments should be provided for every block of code. These should explain what the block of code is aiming to achieve in plain english.
  - Inline comments should specify any non-trivial caveats with a given piece of code, considerations to make when maintaining it, etc.
  - Any considerations regarding potential weaknesses or future improvements to be made within your code should be accompanied by an inline comment prefixed with `// TODO: `
  - Comments should provide some value to a new contributor and improve code readability.
- Your changes MUST NOT contain copyrighted or licensed works without ensuring copyright law or licenses have not been violated.

If any of these requirements are violated, you should expect your pull request to be denied merging until appropriately remediated. 

Pull request reviewers have a responsibility to uphold these standards. Even if a pull request is compliant with these requirements, a reviewer which identifies an opportunity to document some caveat (such as a `// TODO: ` comment) should request it be added prior to pull request approval.

## Considerations for cross-platform code
- While Linux/macOS us LF line endings (`\n`), Windows uses CRLF line endings (`\r\n`). This is important to note when processing text files (e.g., performing source mapping operations).
- Windows file paths use backslashes (`\\`) while other operating systems tend to use forward-slashes (`/`). Many APIs will support forward slashes on Windows, but some may not. This is generally important to note when constructing file paths or performing equality checks. Ensure file paths are normalized for consistency where possible.

## License
TODO
