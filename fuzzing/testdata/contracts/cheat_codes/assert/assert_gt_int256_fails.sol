// This test ensures that assertGt(int256,int256) fails when a <= b
interface CheatCodes {
    function assertGt(int256, int256) external;
}

contract TestContract {
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function test_assertGt_int256_fails() public {
        cheats.assertGt(int256(-5), int256(5));
    }
}
