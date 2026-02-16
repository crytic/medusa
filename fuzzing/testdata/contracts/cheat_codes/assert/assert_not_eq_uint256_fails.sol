// This test ensures that assertNotEq(uint256,uint256) fails with equal values
interface CheatCodes {
    function assertNotEq(uint256, uint256) external;
}

contract TestContract {
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function test_assertNotEq_uint256_fails() public {
        cheats.assertNotEq(uint256(42), uint256(42));
    }
}
