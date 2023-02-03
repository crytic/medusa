interface CheatCodes {
    function warp(uint256) external;
}

contract TestContract {
    function test(uint256 time) public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Back up the original value.
        uint original = block.timestamp;

        // Change to the provided value and verify.
        cheats.warp(time);
        assert(block.timestamp == time);

        // Change back to the original value and verify.
        cheats.warp(original);
        assert(block.timestamp == original);
    }
}
