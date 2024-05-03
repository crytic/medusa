# `ffi`

## Description

The `ffi` cheatcode is used to call an arbitrary command on your host OS. Note that `ffi` must be enabled via the project
configuration file by setting `fuzzing.chainConfig.cheatCodes.enableFFI` to `true`.

Note that enabling `ffi` allows anyone to execute arbitrary commands on devices that run the fuzz tests which may
become a security risk.

Please review [Foundry's documentation on the `ffi` cheatcode](https://book.getfoundry.sh/cheatcodes/ffi#tips) for general tips.

## Example with ABI-encoded hex

```solidity
// Obtain our cheat code contract reference.
IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

// Create command
string[] memory inputs = new string[](3);
inputs[0] = "echo";
inputs[1] = "-n";
// Encoded "hello"
inputs[2] = "0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000568656C6C6F000000000000000000000000000000000000000000000000000000";

// Call cheats.ffi
bytes memory res = cheats.ffi(inputs);

// ABI decode
string memory output = abi.decode(res, (string));
assert(keccak256(abi.encodePacked(output)) == keccak256(abi.encodePacked("hello")));
```

## Example with UTF8 encoding

```solidity
// Obtain our cheat code contract reference.
IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

// Create command
string[] memory inputs = new string[](3);
inputs[0] = "echo";
inputs[1] = "-n";
inputs[2] = "hello";

// Call cheats.ffi
bytes memory res = cheats.ffi(inputs);

// Convert to UTF-8 string
string memory output = string(res);
assert(keccak256(abi.encodePacked(output)) == keccak256(abi.encodePacked("hello")));
```

## Function Signature

```solidity
function ffi(string[] calldata) external returns (bytes memory);
```
