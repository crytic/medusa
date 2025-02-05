# `difficulty`

## Description

The `difficulty` cheatcode has been deprecated in `medusa`. Since `medusa` uses a post-Paris EVM version, the cheatcode
will not update the `block.difficulty` and instead calling it will be a no-op. 

## Function Signature

```solidity
function difficulty(uint256) external;
```
