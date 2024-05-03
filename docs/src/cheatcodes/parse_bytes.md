# `parseBytes`

## Description

The `parseBytes` cheatcode will parse the input string into bytes

## Example

```solidity
contract TestContract {
    uint x = 123;
    function test() public {
        // Obtain our cheat code contract reference.
        IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        bytes memory expectedBytes = "medusa";
        string memory test = "medusa";

        // Call cheats.parseBytes
        bytes memory result = cheats.parseBytes(test);
        assert(keccak256(expectedBytes) == keccak256(result));
    }
}
```

## Function Signature

```solidity
function parseBytes(string calldata) external returns (bytes memory);
```
