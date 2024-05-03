# `getNonce`

## Description

The `getNonce` cheatcode will get the current nonce of `account`.

## Example

```solidity
// Obtain our cheat code contract reference.
IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

// Get nonce and verify that the sender has sent at least one transaction
address acc = address(msg.sender);
assert(cheats.getNonce(acc) > 0);
```

## Function Signature

```solidity
function getNonce(address account) external returns (uint64);
```
