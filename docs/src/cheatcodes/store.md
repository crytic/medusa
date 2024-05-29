# `store`

## Description

The `store` cheatcode will store `value` in storage slot `slot` for `account`

## Example

```solidity
contract TestContract {
    uint x = 123;
    function test() public {
        // Obtain our cheat code contract reference.
        IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Store into x, verify it.
        cheats.store(address(this), bytes32(uint(0)), bytes32(uint(456)));
        assert(y == 456);
    }
}
```

## Function Signature

```solidity
function store(address account, bytes32 slot, bytes32 value) external;
```
