# `addr`

## Description

The `addr` cheatcode will compute the address for a given private key.

## Example

```solidity
// Obtain our cheat code contract reference.
IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

// Test with random private key
uint256 pkOne = 0x6df21769a2082e03f7e21f6395561279e9a7feb846b2bf740798c794ad196e00;
address addrOne = 0xdf8Ef652AdE0FA4790843a726164df8cf8649339;
address result = cheats.addr(pkOne);
assert(result == addrOne);
```

## Function Signature

```solidity
function addr(uint256 privateKey) external returns (address);
```
