// This test ensures that assertion cheatcodes work correctly
interface CheatCodes {
    function assertTrue(bool) external;
    function assertFalse(bool) external;

    function assertEq(bool, bool) external;
    function assertEq(uint256, uint256) external;
    function assertEq(int256, int256) external;
    function assertEq(address, address) external;
    function assertEq(bytes32, bytes32) external;
    function assertEq(string calldata, string calldata) external;
    function assertEq(bytes calldata, bytes calldata) external;

    function assertNotEq(bool, bool) external;
    function assertNotEq(uint256, uint256) external;
    function assertNotEq(int256, int256) external;
    function assertNotEq(address, address) external;
    function assertNotEq(bytes32, bytes32) external;
    function assertNotEq(string calldata, string calldata) external;
    function assertNotEq(bytes calldata, bytes calldata) external;

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
    // All of thse should pass, and should *not* trigger assertion failure
    function test() public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Test assertTrue and assertFalse
        cheats.assertTrue(true);
        cheats.assertFalse(false);

        // Test assertEq for different types
        cheats.assertEq(true, true);
        cheats.assertEq(uint256(42), uint256(42));
        cheats.assertEq(int256(-42), int256(-42));
        cheats.assertEq(address(0x1234), address(0x1234));
        cheats.assertEq(bytes32(uint256(123)), bytes32(uint256(123)));
        cheats.assertEq(string("hello"), string("hello"));
        cheats.assertEq(bytes(hex"aabb"), bytes(hex"aabb"));

        // Test assertNotEq for different types
        cheats.assertNotEq(true, false);
        cheats.assertNotEq(uint256(42), uint256(43));
        cheats.assertNotEq(int256(-42), int256(-43));
        cheats.assertNotEq(address(0x1234), address(0x5678));
        cheats.assertNotEq(bytes32(uint256(123)), bytes32(uint256(456)));
        cheats.assertNotEq(string("hello"), string("world"));
        cheats.assertNotEq(bytes(hex"aabb"), bytes(hex"ccdd"));

        // Test comparison assertions
        cheats.assertLt(uint256(5), uint256(10));
        cheats.assertLt(int256(-10), int256(5));

        cheats.assertLe(uint256(5), uint256(10));
        cheats.assertLe(uint256(5), uint256(5));
        cheats.assertLe(int256(-10), int256(5));
        cheats.assertLe(int256(-10), int256(-10));

        cheats.assertGt(uint256(10), uint256(5));
        cheats.assertGt(int256(5), int256(-10));

        cheats.assertGe(uint256(10), uint256(5));
        cheats.assertGe(uint256(10), uint256(10));
        cheats.assertGe(int256(5), int256(-10));
        cheats.assertGe(int256(5), int256(5));
    }

    // All of the following should revert with an assertion panic
    function testFailAsserttrue() public {
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
        cheats.assertTrue(false);
    }

    function testFailAssertEq() public {
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
        cheats.assertEq(uint256(1), uint256(2));
    }

    function testFailAssertLt() public {
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
        cheats.assertLt(uint256(10), uint256(5));
    }
}
