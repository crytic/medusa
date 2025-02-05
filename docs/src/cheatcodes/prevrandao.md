# `prevrandao`

## Description

The `prevrandao` cheatcode updates the `block.prevrandao`. 

## Example

```solidity
// Obtain our cheat code contract reference.
IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

// Change value and verify.
cheats.prevrandao(bytes32(uint256(42)));
assert(block.prevrandao == 42);
```

## Function Signature

```solidity
function prevrandao(bytes32) external;
```
