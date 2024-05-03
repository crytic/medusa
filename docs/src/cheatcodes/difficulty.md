# `difficulty`

## Description

The `difficulty` cheatcode will set the `block.difficulty` and the `block.prevrandao` value. At the moment, both values
are changed since the cheatcode does not check what EVM version is running.

Note that this behavior will change in the future.

## Example

```solidity
// Obtain our cheat code contract reference.
IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

// Change value and verify.
cheats.difficulty(x);
assert(block.difficulty == x);
```

## Function Signature

```solidity
function difficulty(uint256) external;
```
