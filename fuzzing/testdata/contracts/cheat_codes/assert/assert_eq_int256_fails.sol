// This test ensures that assertEq(int256,int256) fails with different values
interface CheatCodes {
    function assertEq(int256, int256) external;
}

contract TestContract {
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function test_assertEq_int256_fails() public {
        cheats.assertEq(int256(42), int256(-42));
    }
}
