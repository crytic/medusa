// This test ensures that assertNotEq(bool,bool) fails with equal values
interface CheatCodes {
    function assertNotEq(bool, bool) external;
}

contract TestContract {
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function test_assertNotEq_bool_fails() public {
        cheats.assertNotEq(true, true);
    }
}
