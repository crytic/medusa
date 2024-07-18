# `fee`

## Description

The `fee` cheatcode will set the `block.basefee`.

## Example

```solidity
// Obtain our cheat code contract reference.
IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

// Change value and verify.
cheats.fee(7);
assert(block.basefee == 7);
```

## Function Signature

```solidity
function fee(uint256) external;
```
