// This test ensures that assertEq(uint256,uint256) fails with different values
interface CheatCodes {
    function assertEq(uint256, uint256) external;
}

contract TestContract {
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function test_assertEq_uint256_fails() public {
        cheats.assertEq(uint256(42), uint256(43));
    }
}
