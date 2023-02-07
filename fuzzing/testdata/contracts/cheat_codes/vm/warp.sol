// This test ensures that the block timestamp can be set with cheat codes
interface CheatCodes {
    function warp(uint256) external;
}

contract TestContract {
    function test(uint256 time) public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Change value and verify.
        cheats.warp(time);
        assert(block.timestamp == time);
        cheats.warp(7);
        assert(block.timestamp == 7);
    }
}
