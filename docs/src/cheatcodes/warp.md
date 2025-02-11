# warp

## Description

The `warp` cheatcode sets the `block.timestamp`

## Example

```solidity
// Obtain our cheat code contract reference.
IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

// Change value and verify.
cheats.warp(7);
assert(block.timestamp == 7);
cheats.warp(9);
assert(block.timestamp == 9);
```

## Function Signature

```solidity
function warp(uint256) external;
```
