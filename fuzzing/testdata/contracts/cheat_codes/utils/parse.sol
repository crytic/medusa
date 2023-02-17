interface CheatCodes {
    function parseBytes(string calldata) external returns (bytes memory);
    function parseBytes32(string calldata) external returns (bytes32);
    function parseAddress(string calldata) external returns (address);
    function parseUint(string calldata) external returns (uint256);
    function parseInt(string calldata) external returns (int256);
    function parseBool(string calldata) external returns (bool);
}

contract TestContract {
    CheatCodes cheats;

    constructor() {
        cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
    }

    function testAddress() public {
        address expectedAddress = 0x7109709ECfa91a80626fF3989D68f67F5b1DD12D;
        string memory test = "0x7109709ECfa91a80626fF3989D68f67F5b1DD12D";

        // Call cheats.parseAddress
        address result = cheats.parseAddress(test);
        assert(expectedAddress == result);
    }

    function testBool() public {
        bool expectedBool = true;
        string memory test = "true";

        // Call cheats.parseBool
        bool result = cheats.parseBool(test);
        assert(expectedBool == result);
    }

    function testUint256() public {
        uint256 expectedUint = 12345;
        string memory test = "12345";

        // Call cheats.parseUint
        uint256 result = cheats.parseUint(test);
        assert(expectedUint == result);
    }

    function testInt256() public {
        int256 expectedInt = -12345;
        string memory test = "-12345";

        // Call cheats.parseInt
        int256 result = cheats.parseInt(test);
        assert(expectedInt == result);
    }

    function testBytes32() public {
        bytes32 expectedBytes32 = "medusa";
        string memory test = "medusa";

        // Call cheats.parseBytes32
        bytes32 result = cheats.parseBytes32(test);
        assert(expectedBytes32 == result);
    }

    function testBytes() public {
        bytes memory expectedBytes = "medusa";
        string memory test = "medusa";

        // Call cheats.parseBytes
        bytes memory result = cheats.parseBytes(test);
        assert(keccak256(expectedBytes) == keccak256(result));
    }

}
