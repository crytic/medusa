// This test ensures that assertTrue fails when condition is false
interface CheatCodes {
    function assertTrue(bool) external;
}

contract TestContract {
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function test_assertTrue_fails(bool condition) public {
        cheats.assertTrue(condition);
    }
}
