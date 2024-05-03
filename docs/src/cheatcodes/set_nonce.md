# setNonce

## Description

The `setNonce` cheatcode will set the nonce of `account` to `nonce`. Note that the `nonce` must be strictly greater than
the current nonce

## Example

```solidity
// Obtain our cheat code contract reference.
IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

// Set nonce and verify (assume nonce before `setNonce` was less than 7)
address acc = address(msg.sender);
cheats.setNonce(acc, 7);
assert(cheats.getNonce(acc) == 7);
```

## Function Signature

```solidity
function setNonce(address account, uint64 nonce) external;
```
