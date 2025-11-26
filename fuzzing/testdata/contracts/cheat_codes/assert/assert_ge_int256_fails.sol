// This test ensures that assertGe(int256,int256) fails when a < b
interface CheatCodes {
    function assertGe(int256, int256) external;
}

contract TestContract {
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function test_assertGe_int256_fails() public {
        cheats.assertGe(int256(-10), int256(0));
    }
}
