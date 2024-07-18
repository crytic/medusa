# `load`

## Description

The `load` cheatcode will load storage slot `slot` for `account`

## Example

```solidity
contract TestContract {
    uint x = 123;
    function test() public {
        // Obtain our cheat code contract reference.
        IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Load and verify x
        bytes32 value = cheats.load(address(this), bytes32(uint(0)));
        assert(value == bytes32(uint(123)));
    }
}
```

## Function Signature

```solidity
function load(address account, bytes32 slot) external returns (bytes32);
```
