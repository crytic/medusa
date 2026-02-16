// This test ensures that assertLe(uint256,uint256) fails when a > b
interface CheatCodes {
    function assertLe(uint256, uint256) external;
}

contract TestContract {
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function test_assertLe_uint256_fails() public {
        cheats.assertLe(uint256(100), uint256(50));
    }
}
