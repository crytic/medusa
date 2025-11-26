// This test ensures that assertEq(string,string) fails with different values
interface CheatCodes {
    function assertEq(string memory, string memory) external;
}

contract TestContract {
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function test_assertEq_string_fails() public {
        string memory s1 = "hello";
        string memory s2 = "world";
        cheats.assertEq(s1, s2);
    }
}
