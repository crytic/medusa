# `deal`

## Description

The `deal` cheatcode will set the ETH balance of address `who` to `newBalance`

## Example

```solidity
// Obtain our cheat code contract reference.
IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

// Change value and verify.
address acc = address(777);
cheats.deal(acc, x);
assert(acc.balance == x);
```

## Function Signature

```solidity
function deal(address who, uint256 newBalance) external;
```
