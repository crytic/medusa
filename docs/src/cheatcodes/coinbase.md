# `coinbase`

## Description

The `coinbase` cheatcode will set the `block.coinbase`

## Example

```solidity
// Obtain our cheat code contract reference.
IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

// Change value and verify.
cheats.coinbase(address(7));
assert(block.coinbase == address(7));
```

## Function Signature

```solidity
function coinbase(address) external;
```
