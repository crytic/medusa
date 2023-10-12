# Description

This repository contains fixes to a number issues from the [medusa](https://github.com/crytic/medusa) repo.

## Fixed Issues

### 1. Fail On Revert ([Issue 192](https://github.com/crytic/medusa/issues/192))

`The problem`: Currently, Medusa does not provide an option for assertion tests to fail on reverts. One problem with this is that users then have to rely on coverage reports to debug and figure out what went wrong, and where. Furthermore, implementing a fail on revert will enforce that developers are more careful with their preconditions and scoping of testing.

`Our solution` ([fail_on_revert](https://github.com/brainycodelab/medusa-fork/tree/fail_on_revert) Branch): In order not to impose behaviour on everyone, we introduced a new option to the medusa fuzzer `AssertionModesConfig` called `FailOnRevert` which is disabled by default.

```go
// AssertionModesConfig describes the configuration options for the various modes that can be enabled for assertion
// testing
type AssertionModesConfig struct {
    // FailOnCompilerInsertedPanic describes whether a generic compiler inserted panic should be treated as a failing case
  FailOnCompilerInsertedPanic bool `json:"failOnCompilerInsertedPanic"`

  // FailOnAssertion describes whether an assertion failure should be treated as a failing case
  FailOnAssertion bool `json:"failOnAssertion"`

  // FailOnRevert describes whether a revert should be treated as a failing case
  FailOnRevert bool `json:"failOnRevert"`

  // FailOnArithmeticUnderflow describes whether an arithmetic underflow should be treated as a failing case
  FailOnArithmeticUnderflow bool `json:"failOnArithmeticUnderflow"`

  // FailOnDivideByZero describes whether division by zero should be treated as a failing case
  FailOnDivideByZero bool `json:"failOnDivideByZero"`

  // FailOnEnumTypeConversionOutOfBounds describes whether an out-of-bounds enum access should be treated as a failing case
  FailOnEnumTypeConversionOutOfBounds bool `json:"failOnEnumTypeConversionOutOfBounds"`

  // FailOnIncorrectStorageAccess describes whether an out-of-bounds storage access should be treated as a failing case
  FailOnIncorrectStorageAccess bool `json:"failOnIncorrectStorageAccess"`

  // FailOnPopEmptyArray describes whether a pop operation on an empty array should be treated as a failing case
  FailOnPopEmptyArray bool `json:"failOnPopEmptyArray"`

  // FailOnOutOfBoundsArrayAccess describes whether an out-of-bounds array access should be treated as a failing case
  FailOnOutOfBoundsArrayAccess bool `json:"failOnOutOfBoundsArrayAccess"`

  // FailOnAllocateTooMuchMemory describes whether excessive memory usage should be treated as a failing case
  FailOnAllocateTooMuchMemory bool `json:"failOnAllocateTooMuchMemory"`

  // FailOnCallUninitializedVariable describes whether calling an un-initialized variable should be treated as a failing case
  FailOnCallUninitializedVariable bool `json:"failOnCallUninitializedVariable"`
}
```

  <br />

When enabled, if an assertion test encounters a `revert` or `require` error, the assertion test fails. This is implemented using a few lines of code in the `checkAssertionFailures` function of the `test_case_assertion_provider.go` file.

  <br />

```go
  // Check for revert or require failures if FailOnRevert is set to true
  if t.fuzzer.config.Fuzzing.Testing.AssertionTesting.AssertionModes.FailOnRevert {
    if lastExecutionResult.Err == vm.ErrExecutionReverted {
      return &methodId, true, nil
    }
  }
```

### 2. Open all non-zero coverage reports by default ([Issue 233](https://github.com/crytic/medusa/issues/233))

`The problem`: Currently, all coverage report accordions in the medusa coverage report html file are collapsed by default. During debugging, a user will usually have to go back and forth between fuzzing and analyzing coverage reports. Upon refreshing the coverage reports page, all accordions will get closed again, so the user has to manually open the coverage reports they're analyzing every single time. If all coverage reports with more than 0% coverage are open by default, a convenient way to find what you're looking for on the page will be to use `Ctrl + F`.

`Our solution` ([open-corpus-report-if-greater-than-0](https://github.com/brainycodelab/medusa-fork/tree/open-corpus-report-if-greater-than-0) Branch): We updated the gohtml template used to generate the coverage reports to render an already open coverage report accordion if the coverage is greater than 0%.

```html
{{if not $linesCoveredPercentInt}}
<button class="collapsible">
  {{/*The progress bar's color is set from HSL values (hue 0-100 is
  red->orange->yellow->green)*/}}
  <span
    ><progress
      class="progress-coverage"
      value="{{percentageStr $linesCovered $linesActive 0}}"
      max="100"
      style="accent-color: hsl({{$linesCoveredPercentInt}}, 100%, 60%)"
    ></progress
  ></span>
  <span>[{{percentageStr $linesCovered $linesActive 0}}%]</span>
  <span>{{relativePath $sourceFile.Path}}</span>
</button>
{{else}}
<button class="collapsible collapsible-active">
  {{/*The progress bar's color is set from HSL values (hue 0-100 is
  red->orange->yellow->green)*/}}
  <span
    ><progress
      class="progress-coverage"
      value="{{percentageStr $linesCovered $linesActive 0}}"
      max="100"
      style="accent-color: hsl({{$linesCoveredPercentInt}}, 100%, 60%)"
    ></progress
  ></span>
  <span>[{{percentageStr $linesCovered $linesActive 0}}%]</span>
  <span>{{relativePath $sourceFile.Path}}</span>
</button>
{{end}}
```

<br />

Also, currently, javascript is used to conditionally show each coverage report content on click of the coverage report button:

<br />

```js
// Add event listeners for collapsible sections to collapse/expand on click.
const collapsibleHeaders = document.getElementsByClassName('collapsible');
let i;
for (i = 0; i < collapsibleHeaders.length; i++) {
  collapsibleHeaders[i].addEventListener('click', function () {
    this.classList.toggle('collapsible-active');
    const collapsibleContainer = this.nextElementSibling;
    if (collapsibleContainer.style.maxHeight) {
      collapsibleContainer.style.maxHeight = null;
    } else {
      collapsibleContainer.style.maxHeight =
        collapsibleContainer.scrollHeight + 'px';
    }
  });
}

// If there's only one item, expand it by default.
if (collapsibleHeaders.length === 1) {
  collapsibleHeaders[0].click();
}
```

<br />

We've simplified this by only using javascript where necessary:

<br />

```js
// Add event listeners for collapsible sections to collapse/expand on click.
const collapsibleHeaders = document.getElementsByClassName('collapsible');
let i;
for (i = 0; i < collapsibleHeaders.length; i++) {
  collapsibleHeaders[i].addEventListener('click', function () {
    this.classList.toggle('collapsible-active');
  });
}

// If there's only one item and that item has 0% coverage, expand it by default.
if (
  collapsibleHeaders.length === 1 &&
  !collapsibleHeaders.className.contains('collapsible-active')
) {
  collapsibleHeaders[0].click();
}
```

<br />

And using CSS in its place to display coverage report content

<br />

```css
.collapsible-active + .collapsible-container {
  max-height: none;
}
```
