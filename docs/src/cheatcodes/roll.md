# `roll`

## Description

The `roll` cheatcode sets the `block.number`

## Example

```solidity
// Obtain our cheat code contract reference.
IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

// Change value and verify.
cheats.roll(7);
assert(block.number == 7);
cheats.roll(9);
assert(block.number == 9);
```

## Function Signature

```solidity
function roll(uint256) external;
```
