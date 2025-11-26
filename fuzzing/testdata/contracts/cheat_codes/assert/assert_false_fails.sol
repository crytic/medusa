// This test ensures that assertFalse fails when condition is true
interface CheatCodes {
    function assertFalse(bool) external;
}

contract TestContract {
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function test_assertFalse_fails(bool condition) public {
        cheats.assertFalse(condition);
    }
}
