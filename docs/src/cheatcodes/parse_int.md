# `parseInt`

## Description

The `parseInt` cheatcode will parse the input string into a int256

## Example

```solidity
contract TestContract {
    uint x = 123;
    function test() public {
        // Obtain our cheat code contract reference.
        IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        address expectedAddress = 0x7109709ECfa91a80626fF3989D68f67F5b1DD12D;
        string memory test = "0x7109709ECfa91a80626fF3989D68f67F5b1DD12D";

        // Call cheats.parseAddress
        address result = cheats.parseAddress(test);
        assert(expectedAddress == result);
    }
}
```

## Function Signature

```solidity
function parseInt(string calldata) external returns (int256);
```
