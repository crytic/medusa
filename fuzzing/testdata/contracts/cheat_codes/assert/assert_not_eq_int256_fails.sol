// This test ensures that assertNotEq(int256,int256) fails with equal values
interface CheatCodes {
    function assertNotEq(int256, int256) external;
}

contract TestContract {
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function test_assertNotEq_int256_fails() public {
        cheats.assertNotEq(int256(-42), int256(-42));
    }
}
