// This test ensures that assertEq(address,address) fails with different values
interface CheatCodes {
    function assertEq(address, address) external;
}

contract TestContract {
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function test_assertEq_address_fails() public {
        cheats.assertEq(address(0x1234), address(0x5678));
    }
}
