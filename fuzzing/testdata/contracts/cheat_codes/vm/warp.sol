interface CheatCodes {
    // Set block.timestamp (newTimestamp)
    function warp(uint256) external;
}

contract TestContract {
    function testWarp(uint256 time) public {
        // Obtain our cheat code contract reference.
        CheatCodes cheatCodes = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Back up the original time
        uint originalTime = block.timestamp;

        // Warp to requested time and verify it worked
        cheatCodes.warp(time);
        assert(block.timestamp == time);

        // Warp back to original time and verify it worked.
        cheatCodes.warp(originalTime);
        assert(block.timestamp == originalTime);
    }
}
