# Testing with `medusa`

`medusa`, like Echidna, supports the following testing modes:

1. [Property Mode](https://secure-contracts.com/program-analysis/echidna/introduction/how-to-test-a-property.html)
2. [Assertion Mode](https://secure-contracts.com/program-analysis/echidna/basic/assertion-checking.html)
3. [Optimization Mode](https://secure-contracts.com/program-analysis/echidna/advanced/optimization_mode.html)

For more advanced information and documentation on how the various modes work and their pros/cons, check out [secure-contracts.com](https://secure-contracts.com/program-analysis/echidna/index.html)

## Writing property tests

Property tests are represented as functions within a Solidity contract whose names are prefixed with a prefix specified by the `testPrefixes` configuration option (`fuzz_` is the default test prefix). Additionally, they must take no arguments and return a `bool` indicating if the test succeeded.

```solidity
contract TestXY {
    uint x;
    uint y;

    function setX(uint value) public {
        x = value + 3;
    }

    function setY(uint value) public {
        y = value + 9;
    }

    function fuzz_never_specific_values() public returns (bool) {
        // ASSERTION: x should never be 10 at the same time y is 80
        return !(x == 10 && y == 80);
    }
}
```

`medusa` deploys your contract containing property tests and generates a sequence of calls to execute against all publicly accessible methods. After each function call, it calls upon your property tests to ensure they return a `true` (success) status.

### Testing in property-mode

To begin a fuzzing campaign in property-mode, you can run `medusa fuzz` or `medusa fuzz --config [config_path]`.

> **Note**: Learn more about running `medusa` with its CLI [here](../cli/overview.md).

Invoking this fuzzing campaign, `medusa` will:

- Compile the given targets
- Start the configured number of worker threads, each with their own local Ethereum test chain.
- Deploy all contracts to each worker's test chain.
- Begin to generate and send call sequences to update contract state.
- Check property tests all succeed after each call executed.

Upon discovery of a failed property test, `medusa` will halt, reporting the call sequence used to violate any property test(s):

```
[FAILED] Property Test: TestXY.fuzz_never_specific_values()
Test "TestXY.fuzz_never_specific_values()" failed after the following call sequence:
1) TestXY.setY([71]) (gas=4712388, gasprice=1, value=0, sender=0x2222222222222222222222222222222222222222)
2) TestXY.setX([7]) (gas=4712388, gasprice=1, value=0, sender=0x3333333333333333333333333333333333333333)
```

## Writing assertion tests

Although both property-mode and assertion-mode try to validate / invalidate invariants of the system, they do so in different ways. In property-mode, `medusa` will look for functions with a specific test prefix (e.g. `fuzz_`) and test those. In assertion-mode, `medusa` will test to see if a given call sequence can cause the Ethereum Virtual Machine (EVM) to "panic". The EVM has a variety of panic codes for different scenarios. For example, there is a unique panic code when an `assert(x)` statement returns `false` or when a division by zero is encountered. In assertion mode, which panics should or should not be treated as "failing test cases" can be toggled by updating the [Project Configuration](../project_configuration/fuzzing_config.md#fuzzing-configuration). By default, only `FailOnAssertion` is enabled. Check out the [Example Project Configuration File](https://github.com/crytic/medusa/wiki/Example-Project-Configuration-File) for a visualization of the various panic codes that can be enabled. An explanation of the various panic codes can be found in the [Solidity documentation](https://docs.soliditylang.org/en/latest/control-structures.html#panic-via-assert-and-error-via-require).

Please note that the behavior of assertion mode is different between `medusa` and Echidna. Echidna will only test for `assert(x)` statements while `medusa` provides additional flexibility.

```solidity
contract TestContract {
    uint x;
    uint y;

    function setX(uint value) public {
        x = value;

        // ASSERTION: x should be an even number
        assert(x % 2 == 0);
    }
}
```

During a call sequence, if `setX` is called with a `value` that breaks the assertion (e.g. `value = 3`), `medusa` will treat this as a failing property and report it back to the user.

### Testing in assertion-mode

To begin a fuzzing campaign in assertion-mode, you can run `medusa fuzz --assertion-mode` or `medusa fuzz --config [config_path] --assertion-mode`.

> **Note**: Learn more about running `medusa` with its CLI [here](../cli/overview.md).

Invoking this fuzzing campaign, `medusa` will:

- Compile the given targets
- Start the configured number of worker threads, each with their own local Ethereum test chain.
- Deploy all contracts to each worker's test chain.
- Begin to generate and send call sequences to update contract state.
- Check to see if there any failing assertions after each call executed.

Upon discovery of a failed assertion, `medusa` will halt, reporting the call sequence used to violate any assertions:

```
Fuzzer stopped, test results follow below ...
[FAILED] Assertion Test: TestContract.setX(uint256)
Test for method "TestContract.setX(uint256)" failed after the following call sequence resulted in an assertion:
1) TestContract.setX([102552480437485684723695021980667056378352338398148431990087576385563741034353]) (block=2, time=4, gas=12500000, gasprice=1, value=0, sender=0x1111111111111111111111111111111111111111)
```

## Writing optimization tests

Optimization mode's goal is not to validate/invalidate properties but instead to maximize the return value of a function. Similar to property mode, these functions must be prefixed with a prefix specified by the `testPrefixes` configuration option (`optimize_` is the default test prefix). Additionally, they must take no arguments and return an `int256`. A good use case for optimization mode is to try to quantify the impact of a bug (e.g. a rounding error).

```solidity
contract TestContract {
    int256 input;

    function set(int256 _input) public {
        input = _input;
    }

    function optimize_opt_linear() public view returns (int256) {
        if (input > -4242) return -input;
        else return 0;
    }
}
```

`medusa` deploys your contract containing optimization tests and generates a sequence of calls to execute against all publicly accessible methods. After each function call, it calls upon your otpimization tests to identify whether the return value of those tests are greater than the currently stored values.

### Testing in optimization-mode

To begin a fuzzing campaign in optimization-mode, you can run `medusa fuzz --optimization-mode` or `medusa fuzz --config [config_path] --optimization-mode`.

> **Note**: Learn more about running `medusa` with its CLI [here](../cli/overview.md).

Invoking this fuzzing campaign, `medusa` will:

- Compile the given targets
- Start the configured number of worker threads, each with their own local Ethereum test chain.
- Deploy all contracts to each worker's test chain.
- Begin to generate and send call sequences to update contract state.
- Check to see if the return value of the optimization test is greater than the cached value.
  - If the value is greater, update the cached value.

Once the test limit or timeout for the fuzzing campaign has been reached, `medusa` will halt and report the call sequence that maximized the return value of the function:

```
Fuzzer stopped, test results follow below ...
[PASSED] Optimization Test: TestContract.optimize_opt_linear()
Optimization test "TestContract.optimize_opt_linear()" resulted in the maximum value: 4241 with the following sequence:
1) TestContract.set(-4241) (block=2, time=3, gas=12500000, gasprice=1, value=0, sender=0x0000000000000000000000000000000000010000)
```

## Testing with multiple modes

Note that we can run `medusa` with one, many, or no modes enabled. Running `medusa fuzz --assertion-mode --optimization-mode` will run all three modes at the same time, since property-mode is enabled by default. If a project configuration file is used, any combination of the three modes can be toggled. In fact, all three modes can be disabled and `medusa` will still run. Please review the [Project Configuration](https://github.com/crytic/medusa/wiki/Project-Configuration) wiki page and the [Project Configuration Example](https://github.com/crytic/medusa/wiki/Example-Project-Configuration-File) for more information.

```solidity
contract TestContract {
    int256 input;

    function set(int256 _input) public {
        input = _input;
    }

    function failing_assert_method(uint value) public {
        // ASSERTION: We always fail when you call this function.
        assert(false);
    }

    function fuzz_failing_property() public view returns (bool) {
        // ASSERTION: fail immediately.
        return false;
    }

    function optimize_opt_linear() public view returns (int256) {
        if (input > -4242) return -input;
        else return 0;
    }
}
```

Invoking a fuzzing campaign with `medusa fuzz --assertion-mode --optimization-mode` (note all three modes are enabled), `medusa` will:

- Compile the given targets
- Start the configured number of worker threads, each with their own local Ethereum test chain.
- Deploy all contracts to each worker's test chain.
- Begin to generate and send call sequences to update contract state.
- Check to see:
  - If property tests all succeed after each call executed.
  - If a panic (which was enabled in the project configuration) has been triggered after each call.
  - Whether the return value of the optimization test is greater than the cached value.
    - Update the cached value if it is greater.
