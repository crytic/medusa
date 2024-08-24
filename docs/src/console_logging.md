# Console Logging

Console logging in medusa is similar to the functionality found in Foundry or Hardhat (except for string formatting,
see [below](#differences-in-consolelogformatargs)). Note that if you are not using
Foundry or Hardhat as your compilation platform, you can retrieve the necessary `console.sol` library
[here](https://github.com/foundry-rs/forge-std/blob/master/src/console.sol).

For more information on the available function signatures and general tips on console logging, please review [Foundry's
documentation](https://book.getfoundry.sh/reference/forge-std/console-log#console-logging).

## Differences in `console.log(format[,...args])`

The core functionality of string formatting is the same. If you want to string format an `int256`, the only supported function signature is:
`function log(string memory, int256) external;`. Otherwise, the supported argument types are `string`, `bool`, `address`,
and `uint256`. This capability is the same as in Foundry.

The core difference in medusa's string formatting is the specifiers that are allowed for the
formatted string. The supported specifiers are as follows:

- `%v`: The value will be printed in its default format. This will work for `uint256`, `int256`, `address`,
  `bool`, and `string`. Using `%v` is the **recommended** specifier for all argument types.
- `%s`: The values will be converted into a human-readable string. This will work for `uint256`, `int256`, `address`, and
  `string`. Contrary to Foundry or Hardhat, `%s` will not work for `bool`. Additionally, `uint256` and `int256` will _not_
  be provided in their hex-encoded format. This is the **recommended** specifier for projects that wish to maintain
  compatibility with an existing fuzz test suite from Foundry. Special exceptions will need to be made for `bool` arguments.
  For example, you could use the `console.logBool(bool)` function to separately log the `bool`.
- `%d`: This can be used for `uint256` and `int256`.
- `%i`: This specifier is not supported by medusa for `int256` and `uint256`
- `%e`: This specifier is not supported by medusa for `int256` and `uint256`.
- `%x`: This provides the hexadecimal representation of `int256` and `uint256`.
- `%o`: This specifier is not supported by medusa. `%o` in medusa will provide the base-8 representation of `int256` and
  `uint256`.
- `%t`: This can be used for `bool`.
- `%%`: This will print out "%" and not consume an argument.

If a specifier does not have a corresponding argument, the following is returned:

```solidity
console.log("My name is %s %s", "medusa");
// Returns: "My name is medusa %!s(MISSING)"
```

If there are more arguments than specifiers, the following is returned:

```solidity
console.log("My name is %s", "medusa", "fuzzer");
// Returns: "My name is medusa%!(EXTRA string=fuzzer)"
```

If only a format string with no arguments is provided, the string is returned with no formatting:

```solidity
console.log("%% %s");
// Returns: "%% %s"
```
