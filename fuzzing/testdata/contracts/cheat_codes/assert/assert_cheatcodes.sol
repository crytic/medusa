// This test ensures that assert cheatcodes work correctly.
// Functions are consolidated to ensure all assertions are executed together.
interface CheatCodes {
    function assertTrue(bool) external;
    function assertFalse(bool) external;
    function assertEq(bool, bool) external;
    function assertEq(uint256, uint256) external;
    function assertEq(int256, int256) external;
    function assertEq(address, address) external;
    function assertEq(bytes32, bytes32) external;
    function assertEq(string memory, string memory) external;
    function assertNotEq(bool, bool) external;
    function assertNotEq(uint256, uint256) external;
    function assertNotEq(int256, int256) external;
    function assertNotEq(address, address) external;
    function assertNotEq(bytes32, bytes32) external;
    function assertNotEq(string memory, string memory) external;
    function assertLt(uint256, uint256) external;
    function assertLt(int256, int256) external;
    function assertLe(uint256, uint256) external;
    function assertLe(int256, int256) external;
    function assertGt(uint256, uint256) external;
    function assertGt(int256, int256) external;
    function assertGe(uint256, uint256) external;
    function assertGe(int256, int256) external;
}

contract TestContract {
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    // Test assertTrue and assertFalse
    function test_boolean_assertions() public {
        cheats.assertTrue(true);
        cheats.assertFalse(false);
    }

    // Test all assertEq variants
    function test_assertEq() public {
        // bool
        cheats.assertEq(true, true);
        cheats.assertEq(false, false);

        // uint256
        cheats.assertEq(uint256(42), uint256(42));
        cheats.assertEq(uint256(0), uint256(0));
        cheats.assertEq(type(uint256).max, type(uint256).max);

        // int256
        cheats.assertEq(int256(42), int256(42));
        cheats.assertEq(int256(-42), int256(-42));
        cheats.assertEq(type(int256).min, type(int256).min);
        cheats.assertEq(type(int256).max, type(int256).max);

        // address
        address addr1 = address(0x1234);
        address addr2 = address(0x1234);
        cheats.assertEq(addr1, addr2);

        // bytes32
        bytes32 b1 = bytes32(uint256(123));
        bytes32 b2 = bytes32(uint256(123));
        cheats.assertEq(b1, b2);

        // string
        cheats.assertEq(string("hello"), string("hello"));
        cheats.assertEq(string(""), string(""));
    }

    // Test all assertNotEq variants
    function test_assertNotEq() public {
        // bool
        cheats.assertNotEq(true, false);
        cheats.assertNotEq(false, true);

        // uint256
        cheats.assertNotEq(uint256(42), uint256(43));
        cheats.assertNotEq(uint256(0), uint256(1));

        // int256
        cheats.assertNotEq(int256(42), int256(43));
        cheats.assertNotEq(int256(-42), int256(42));

        // address
        address addr1 = address(0x1234);
        address addr2 = address(0x5678);
        cheats.assertNotEq(addr1, addr2);

        // bytes32
        bytes32 b1 = bytes32(uint256(123));
        bytes32 b2 = bytes32(uint256(456));
        cheats.assertNotEq(b1, b2);

        // string
        cheats.assertNotEq(string("hello"), string("world"));
        cheats.assertNotEq(string(""), string("test"));
    }

    // Test all comparison assertions (Lt, Le, Gt, Ge)
    function test_comparisons() public {
        // assertLt uint256
        cheats.assertLt(uint256(1), uint256(2));
        cheats.assertLt(uint256(0), uint256(100));

        // assertLt int256
        cheats.assertLt(int256(-10), int256(0));
        cheats.assertLt(int256(-100), int256(-50));
        cheats.assertLt(int256(0), int256(100));

        // assertLe uint256
        cheats.assertLe(uint256(1), uint256(2));
        cheats.assertLe(uint256(5), uint256(5));
        cheats.assertLe(uint256(0), uint256(0));

        // assertLe int256
        cheats.assertLe(int256(-10), int256(0));
        cheats.assertLe(int256(5), int256(5));
        cheats.assertLe(int256(-100), int256(-100));

        // assertGt uint256
        cheats.assertGt(uint256(2), uint256(1));
        cheats.assertGt(uint256(100), uint256(0));

        // assertGt int256
        cheats.assertGt(int256(0), int256(-10));
        cheats.assertGt(int256(-50), int256(-100));
        cheats.assertGt(int256(100), int256(0));

        // assertGe uint256
        cheats.assertGe(uint256(2), uint256(1));
        cheats.assertGe(uint256(5), uint256(5));
        cheats.assertGe(uint256(0), uint256(0));

        // assertGe int256
        cheats.assertGe(int256(0), int256(-10));
        cheats.assertGe(int256(5), int256(5));
        cheats.assertGe(int256(-100), int256(-100));
    }
}
