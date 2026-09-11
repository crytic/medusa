# Testing: Writing Invariants & Running the Suite

This page covers two things: how medusa **tests smart contracts** (the product), and how to **test medusa itself**
(the codebase).

## Part 1 — How medusa tests contracts

medusa detects violations of **invariants** (a.k.a. properties): unchanging truths about a system. It breaks them
into two categories (see [`docs/src/testing/invariants.md`](../docs/src/testing/invariants.md)):

- **Function-level invariants** — properties that must hold from executing a single function.
- **System-level invariants** — properties that must hold across the whole system after arbitrary interactions.

### Test case types

Three built-in test case providers evaluate these invariants (implementation:
[architecture/fuzzing-engine.md](architecture/fuzzing-engine.md)):

| Type             | How you write it                                                                                                            | How it fails                              | Config                                                           |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------- | ---------------------------------------------------------------- |
| **Assertion**    | Use `assert(...)` / cheat-code assertions inside functions                                                                  | An EVM panic with a configured panic code | `testing.assertionTesting.panicCodeConfig`                       |
| **Property**     | No-argument function returning `bool`, named with a prefix (`view`/`pure` by convention — state mutability is not enforced) | Returns `false` (or the call reverts)     | `testing.propertyTesting.testPrefixes` (default `property_`)     |
| **Optimization** | No-argument function returning `int256`, named with a prefix; medusa maximizes the returned value                           | N/A (reports best value + minimal path)   | `testing.optimizationTesting.testPrefixes` (default `optimize_`) |

Authoritative how-to guides (user docs):

- [Writing function-level invariants](../docs/src/testing/writing-function-level-invariants.md)
- [Writing system-level invariants](../docs/src/testing/writing-system-level-invariants.md)
- [Reporting / interpreting results](../docs/src/testing/reporting.md)

### Interpreting results

When a test fails, medusa prints the **shrunk (minimal) call sequence** plus an execution trace (from
[`/fuzzing/executiontracer`](../fuzzing/executiontracer)). Coverage reports (HTML/LCOV) and optional revert reports
are written at the end of the campaign. `testing.verbosity` controls output detail.

## Part 2 — Testing medusa (the Go codebase)

Tests are co-located with implementations as `*_test.go` (see [`/AGENTS.md`](../AGENTS.md) "Testing Guidelines").

```bash
go test -v ./...                       # run all unit + integration tests
go test -v ./fuzzing/...               # run a specific package
go test -v -run TestName ./package/... # run one test
go test -cover ./...                   # with coverage metrics
```

Notable test areas:

- [`fuzzing/fuzzer_test.go`](../fuzzing/fuzzer_test.go) — large end-to-end fuzzer tests using Solidity fixtures in
  [`fuzzing/testdata/contracts`](../fuzzing/testdata/contracts) (assertion tests, property tests, cheat codes, etc.).
- [`chain/test_chain_test.go`](../chain/test_chain_test.go) — TestChain harness (reverting, block jumping, dynamic
  deployments, cloning, call-sequence replay); cheat-code behavior is tested by `TestCheatCodes` in
  [`fuzzing/fuzzer_test.go`](../fuzzing/fuzzer_test.go).
- [`chain/state/*_test.go`](../chain/state) — state factories, remote/fork providers, genesis loading.
- [`compilation/platforms/*_test.go`](../compilation/platforms) — solc/crytic-compile adapters.
- [`fuzzing/config/config_test.go`](../fuzzing/config/config_test.go) — config parsing/defaults.

Conventions: table-driven tests for deterministic logic; add `t.Parallel()` where safe; end-to-end fuzzer tests can
be slow (they compile Solidity and run real campaigns), which is why the full suite is a manual pre-PR check rather
than a pre-commit hook.

### Change-oriented guidance

- Adding a new test case type: add a `test_case_*_provider.go`, register it in `fuzzing/fuzzer.go`, add fixtures
  under `fuzzing/testdata/contracts/`, and add a `TestName` in `fuzzing/fuzzer_test.go`.
- Adding a cheat code: extend `chain/standard_cheat_code_contract.go`, add a fixture under
  `fuzzing/testdata/contracts/cheat_codes/`, and update `docs/src/cheatcodes/`.
- User-facing behavior/config changes: update [`/docs/src`](../docs/src/SUMMARY.md); CI runs
  [`/scripts/check_docs.py`](../scripts/check_docs.py) to verify docs stay in sync.
