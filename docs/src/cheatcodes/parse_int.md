# `parseInt`

## Description

The `parseInt` cheatcode will parse the input string into an int256

## Example

```solidity
contract TestContract {
    uint x = 123;
    function test() public {
        // Obtain our cheat code contract reference.
        IStdCheats cheats = IStdCheats(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        int256 expectedInt = -12345;
        string memory test = "-12345";

        // Call cheats.parseInt
        int256 result = cheats.parseInt(test);
        assert(expectedInt == result);
    }
}
```

## Function Signature

```solidity
function parseInt(string calldata) external returns (int256);
```
