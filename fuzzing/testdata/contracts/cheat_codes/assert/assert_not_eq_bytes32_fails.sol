// This test ensures that assertNotEq(bytes32,bytes32) fails with equal values
interface CheatCodes {
    function assertNotEq(bytes32, bytes32) external;
}

contract TestContract {
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function test_assertNotEq_bytes32_fails() public {
        cheats.assertNotEq(bytes32(uint256(123)), bytes32(uint256(123)));
    }
}
