# `chainId`

## Description

The `chainId` cheatcode will set the `block.chainid`

## Example

```solidity
// Obtain our cheat code contract reference.
IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

// Change value and verify.
cheats.chainId(777123);
assert(block.chainid == 777123);
```

## Function Signature

```solidity
function chainId(uint256) external;
```
