// This test ensures that the block number can be set with cheat codes
interface CheatCodes {
    function roll(uint256) external;
}

contract TestContract {
    function test(uint256 x) public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Change value and verify.
        cheats.roll(x);
        assert(block.number == x);
        cheats.roll(7);
        assert(block.number == 7);
    }
}
