# `parseBytes32`

## Description

The `parseBytes32` cheatcode will parse the input string into bytes32

## Example

```solidity
contract TestContract {
    uint x = 123;
    function test() public {
        // Obtain our cheat code contract reference.
        IStdCheats cheats = IStdCheats(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        bytes32 expectedBytes32 = "medusa";
        string memory test = "medusa";

        // Call cheats.parseBytes32
        bytes32 result = cheats.parseBytes32(test);
        assert(expectedBytes32 == result);
    }
}
```

## Function Signature

```solidity
function parseBytes32(string calldata) external returns (bytes32);
```
