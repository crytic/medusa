interface CheatCodes {
    function toString(address) external returns (string memory);
    function toString(bool) external returns (string memory);
    function toString(uint256) external returns (string memory);
    function toString(int256) external returns (string memory);
    function toString(bytes32) external returns (string memory);
    function toString(bytes memory) external returns (string memory);
}

// Source for test case: https://book.getfoundry.sh/cheatcodes/to-string
contract TestContract {
    CheatCodes cheats;

    constructor() {
        cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
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
