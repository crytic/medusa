// This test ensures that assert cheatcodes work correctly
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

    function test_assertTrue_passes() public {
        cheats.assertTrue(true);
    }

    function test_assertFalse_passes() public {
        cheats.assertFalse(false);
    }

    function test_assertEq_bool() public {
        cheats.assertEq(true, true);
        cheats.assertEq(false, false);
    }

    function test_assertEq_uint256() public {
        cheats.assertEq(uint256(42), uint256(42));
        cheats.assertEq(uint256(0), uint256(0));
        cheats.assertEq(type(uint256).max, type(uint256).max);
    }

    function test_assertEq_int256() public {
        cheats.assertEq(int256(42), int256(42));
        cheats.assertEq(int256(-42), int256(-42));
        cheats.assertEq(type(int256).min, type(int256).min);
        cheats.assertEq(type(int256).max, type(int256).max);
    }

    function test_assertEq_address() public {
        address addr1 = address(0x1234);
        address addr2 = address(0x1234);
        cheats.assertEq(addr1, addr2);
    }

    function test_assertEq_bytes32() public {
        bytes32 b1 = bytes32(uint256(123));
        bytes32 b2 = bytes32(uint256(123));
        cheats.assertEq(b1, b2);
    }

    function test_assertEq_string() public {
        cheats.assertEq(string("hello"), string("hello"));
        cheats.assertEq(string(""), string(""));
    }

    function test_assertNotEq_bool() public {
        cheats.assertNotEq(true, false);
        cheats.assertNotEq(false, true);
    }

    function test_assertNotEq_uint256() public {
        cheats.assertNotEq(uint256(42), uint256(43));
        cheats.assertNotEq(uint256(0), uint256(1));
    }

    function test_assertNotEq_int256() public {
        cheats.assertNotEq(int256(42), int256(43));
        cheats.assertNotEq(int256(-42), int256(42));
    }

    function test_assertNotEq_address() public {
        address addr1 = address(0x1234);
        address addr2 = address(0x5678);
        cheats.assertNotEq(addr1, addr2);
    }

    function test_assertNotEq_bytes32() public {
        bytes32 b1 = bytes32(uint256(123));
        bytes32 b2 = bytes32(uint256(456));
        cheats.assertNotEq(b1, b2);
    }

    function test_assertNotEq_string() public {
        cheats.assertNotEq(string("hello"), string("world"));
        cheats.assertNotEq(string(""), string("test"));
    }

    function test_assertLt_uint256() public {
        cheats.assertLt(uint256(1), uint256(2));
        cheats.assertLt(uint256(0), uint256(100));
    }

    function test_assertLt_int256() public {
        cheats.assertLt(int256(-10), int256(0));
        cheats.assertLt(int256(-100), int256(-50));
        cheats.assertLt(int256(0), int256(100));
    }

    function test_assertLe_uint256() public {
        cheats.assertLe(uint256(1), uint256(2));
        cheats.assertLe(uint256(5), uint256(5));
        cheats.assertLe(uint256(0), uint256(0));
    }

    function test_assertLe_int256() public {
        cheats.assertLe(int256(-10), int256(0));
        cheats.assertLe(int256(5), int256(5));
        cheats.assertLe(int256(-100), int256(-100));
    }

    function test_assertGt_uint256() public {
        cheats.assertGt(uint256(2), uint256(1));
        cheats.assertGt(uint256(100), uint256(0));
    }

    function test_assertGt_int256() public {
        cheats.assertGt(int256(0), int256(-10));
        cheats.assertGt(int256(-50), int256(-100));
        cheats.assertGt(int256(100), int256(0));
    }

    function test_assertGe_uint256() public {
        cheats.assertGe(uint256(2), uint256(1));
        cheats.assertGe(uint256(5), uint256(5));
        cheats.assertGe(uint256(0), uint256(0));
    }

    function test_assertGe_int256() public {
        cheats.assertGe(int256(0), int256(-10));
        cheats.assertGe(int256(5), int256(5));
        cheats.assertGe(int256(-100), int256(-100));
    }
}
