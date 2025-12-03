// This test ensures that assertEq(bool,bool) fails with different values
interface CheatCodes {
    function assertEq(bool, bool) external;
}

contract TestContract {
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function test_assertEq_bool_fails() public {
        cheats.assertEq(true, false);
    }
}
