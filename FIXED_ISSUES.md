# Description

This repository contains fixes to a number issues from the [medusa](https://github.com/crytic/medusa) repo.

## Fixed Issues

### 1. Fail On Revert ([Issue 192](https://github.com/crytic/medusa/issues/192))

`The problem`: Currently, Medusa does not provide an option for assertion tests to fail on reverts. One problem with this is that users then have to rely on coverage reports to debug and figure out what went wrong, and where. Furthermore, implementing a fail on revert will enforce that developers are more careful with their preconditions and scoping of testing.

`Our solution` ([fail_on_revert](https://github.com/brainycodelab/medusa-fork/tree/fail_on_revert) Branch): In order not to impose behaviour on everyone, we introduced a new option to the medusa fuzzer `AssertionModesConfig` called `FailOnRevert` which is disabled by default. When enabled, if an assertion test encounters a `revert` or `require` error, the assertion test fails. This is implemented using a few lines of code in the `checkAssertionFailures` function of the `test_case_assertion_provider.go` file.

### 2. Open all non-zero coverage reports by default ([Issue 233](https://github.com/crytic/medusa/issues/233))

`The problem`: Currently, all coverage report accordions in the medusa coverage report html file are collapsed by default. During debugging, a user will usually have to go back and forth between fuzzing and analyzing coverage reports. Upon refreshing the coverage reports page, all accordions will get closed again, so the user has to manually open the coverage reports they're analyzing every single time. If all coverage reports with more than 0% coverage are open by default, a convenient way to find what you're looking for on the page will be to use `Ctrl + F`.

`Our solution` ([open-corpus-report-if-greater-than-0](https://github.com/brainycodelab/medusa-fork/tree/open-corpus-report-if-greater-than-0) Branch): We updated the gohtml template used to generate the coverage reports to render an already open coverage report accordion if the coverage is greater than 0%.

Also, currently, javascript is used to conditionally show each coverage report content on click of the coverage report button. We've simplified this by only using javascript where necessary and using CSS to conditionally display coverage report content.

And using CSS in its place to display coverage report content

### 3. Medusa fails silently when provided an invalid CLI flag ([Issue 230](https://github.com/crytic/medusa/issues/230))

`The problem`: When provided an invalid CLI flag, medusa fails but doesn't print any error message explaining what went wrong.

`Our solution` ([throw-error-on-invalid-cli-flag](https://github.com/brainycodelab/medusa-fork/tree/throw-error-on-invalid-cli-flag) Branch): We set the `SilenceErrors` flag on the fuzz cobra command to false. This way, whenever an invalid flag is provided, the user gets an error message.

Error message on providing an invalid cli flag:
