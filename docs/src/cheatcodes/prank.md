# `prank`

## Description

The `prank` cheatcode will set the `msg.sender` for _only the next call_ to the specified input address. Note that,
contrary to [`prank` in Foundry](https://book.getfoundry.sh/cheatcodes/prank#description), calling the cheatcode contract will count as a
valid "next call"

## Example

```solidity
contract TestContract {
    address owner = address(123);
    function transferOwnership(address _newOwner) public {
        require(msg.sender == owner);

        // Change ownership
        owner = _newOwner;
    }

    function test() public {
        // Obtain our cheat code contract reference.
        IStdCheats cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Prank, change ownership, and verify
        address newOwner = address(456);
        cheats.prank(owner);
        transferOwnership(newOwner);
        assert(owner == newOwner);
    }
    }
```

## Function Signature

```solidity
function prank(address) external;
```
