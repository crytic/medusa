// This test ensures that assertGt(uint256,uint256) fails when a <= b
interface CheatCodes {
    function assertGt(uint256, uint256) external;
}

contract TestContract {
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function test_assertGt_uint256_fails() public {
        cheats.assertGt(uint256(5), uint256(5));
    }
}
