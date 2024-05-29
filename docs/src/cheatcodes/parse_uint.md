# `parseUint`

## Description

The `parseUint` cheatcode will parse the input string into a uint256

## Example

```solidity
contract TestContract {
    uint x = 123;
    function test() public {
        // Obtain our cheat code contract reference.
        IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        uint256 expectedUint = 12345;
        string memory test = "12345";

        // Call cheats.parseUint
        uint256 result = cheats.parseUint(test);
        assert(expectedUint == result);
    }
}
```

## Function Signature

```solidity
function parseUint(string calldata) external returns (uint256);
```
