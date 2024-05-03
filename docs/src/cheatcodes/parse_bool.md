# `parseBool`

## Description

The `parseBool` cheatcode will parse the input string into a boolean

## Example

```solidity
contract TestContract {
    uint x = 123;
    function test() public {
        // Obtain our cheat code contract reference.
        IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        bool expectedBool = true;
        string memory test = "true";

        // Call cheats.parseBool
        bool result = cheats.parseBool(test);
        assert(expectedBool == result);
    }
}
```

## Function Signature

```solidity
function parseBool(string calldata) external returns (bool);
```
