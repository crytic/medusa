# `etch`

## Description

The `etch` cheatcode will set the `who` address's bytecode to `code`.

## Example

```solidity
// Obtain our cheat code contract reference.
IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

// Obtain our original code hash for an account.
address acc = address(777);
bytes32 originalCodeHash;
assembly { originalCodeHash := extcodehash(acc) }

// Change value and verify.
cheats.etch(acc, address(someContract).code);
bytes32 updatedCodeHash;
assembly { updatedCodeHash := extcodehash(acc) }
assert(originalCodeHash != updatedCodeHash);
```

## Function Signature

```solidity
function etch(address who, bytes calldata code) external;
```
