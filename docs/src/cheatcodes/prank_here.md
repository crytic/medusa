# `prankHere`

## Description

The `prankHere` cheatcode will set the `msg.sender` to the specified input address until the current call exits. Compared
to `prank`, `prankHere` can persist for multiple calls.

## Example

```solidity
contract TestContract {
    address owner = address(123);
    uint256 x = 0;
    uint256 y = 0;

    function updateX() public {
        require(msg.sender == owner);

        // Update x
        x = 1;
    }

    function updateY() public {
        require(msg.sender == owner);

        // Update y
        y = 1;
    }

    function test() public {
        // Obtain our cheat code contract reference.
        IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Prank, update variables, and verify
        cheats.prank(owner);
        updateX();
        updateY();
        assert((x == 1) && (y == 1));

        // Once this function returns, the `msg.sender` is reset
    }
}
```

## Function Signature

```solidity
function prankHere(address) external;
```
