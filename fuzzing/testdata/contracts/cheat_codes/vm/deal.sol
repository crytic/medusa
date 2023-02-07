// This test ensures that account balances can be set with cheat codes
interface CheatCodes {
    function deal(address, uint256) external;
}

contract TestContract {
    function test(uint256 x) public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Change value and verify.
        address acc = address(777);
        cheats.deal(acc, x);
        assert(acc.balance == x);
        cheats.deal(acc, 7 ether);
        assert(acc.balance == 7 ether);
    }
}
