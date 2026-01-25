// This test ensures that assertNotEq(string,string) fails with equal values
interface CheatCodes {
    function assertNotEq(string memory, string memory) external;
}

contract TestContract {
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function test_assertNotEq_string_fails() public {
        string memory s1 = "hello";
        string memory s2 = "hello";
        cheats.assertNotEq(s1, s2);
    }
}
