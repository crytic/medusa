# `parseBytes32`

## Description

The `parseBytes32` cheatcode will parse the input string into bytes32

## Example

```solidity
contract TestContract {
    uint x = 123;
    function test() public {
        // Obtain our cheat code contract reference.
        IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

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
function parseBytes32(string calldata) external returns (bytes32);
```
