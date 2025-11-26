// This test ensures that assertNotEq(address,address) fails with equal values
interface CheatCodes {
    function assertNotEq(address, address) external;
}

contract TestContract {
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function test_assertNotEq_address_fails() public {
        cheats.assertNotEq(address(0x1234), address(0x1234));
    }
}
