# `label`

## Description

The `label` cheatcode sets a label for an address. This is useful for debugging purposes, as labeled addresses will be displayed with their associated labels in execution traces and error messages, making it easier to identify specific contracts or accounts during testing.

## Example

```solidity
// Obtain our cheat code contract reference.
IStdCheats cheats = IStdCheats(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

// Create some addresses
address alice = address(0x1234);
address bob = address(0x5678);

// Label them for easier debugging
cheats.label(alice, "Alice");
cheats.label(bob, "Bob");

// Now when these addresses appear in traces, they'll show as "Alice" and "Bob"
// instead of "0x0000000000000000000000000000000000001234"
```

## Function Signature

```solidity
function label(address account, string calldata label) external;
```
