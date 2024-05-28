# `toString`

## Description

The `toString` cheatcodes aid in converting primitive Solidity types into strings. Similar to
[Foundry's behavior](https://book.getfoundry.sh/cheatcodes/to-string?highlight=toStr#description), bytes are converted
to a hex-encoded string with `0x` prefixed.

## Example

```solidity
contract TestContract {
    IStdCheats cheats;

    constructor() {
        cheats = IStdCheats(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
    }

    function testAddress() public {
        address test = 0x7109709ECfa91a80626fF3989D68f67F5b1DD12D;
        string memory expectedString = "0x7109709ECfa91a80626fF3989D68f67F5b1DD12D";

        // Call cheats.toString
        string memory result = cheats.toString(test);
        assert(keccak256(abi.encodePacked(result)) == keccak256(abi.encodePacked(expectedString)));
    }

    function testBool() public {
        bool test = true;
        string memory expectedString = "true";

        // Call cheats.toString
        string memory result = cheats.toString(test);
        assert(keccak256(abi.encodePacked(result)) == keccak256(abi.encodePacked(expectedString)));
    }

    function testUint256() public {
        uint256 test = 12345;
        string memory expectedString = "12345";

        // Call cheats.toString
        string memory result = cheats.toString(test);
        assert(keccak256(abi.encodePacked(result)) == keccak256(abi.encodePacked(expectedString)));
    }

    function testInt256() public {
        int256 test = -12345;
        string memory expectedString = "-12345";

        // Call cheats.toString
        string memory result = cheats.toString(test);
        assert(keccak256(abi.encodePacked(result)) == keccak256(abi.encodePacked(expectedString)));
    }

    function testBytes32() public {
        bytes32 test = "medusa";
        string memory expectedString = "0x6d65647573610000000000000000000000000000000000000000000000000000";

        // Call cheats.toString
        string memory result = cheats.toString(test);
        assert(keccak256(abi.encodePacked(result)) == keccak256(abi.encodePacked(expectedString)));
    }

    function testBytes() public {
        bytes memory test = "medusa";
        string memory expectedString = "0x6d6564757361";

        // Call cheats.toString
        string memory result = cheats.toString(test);
        assert(keccak256(abi.encodePacked(result)) == keccak256(abi.encodePacked(expectedString)));
    }
}
```

## Function Signatures

```solidity
function toString(address) external returns (string memory);
function toString(bool) external returns (string memory);
function toString(uint256) external returns (string memory);
function toString(int256) external returns (string memory);
function toString(bytes32) external returns (string memory);
function toString(bytes) external returns (string memory);
```
