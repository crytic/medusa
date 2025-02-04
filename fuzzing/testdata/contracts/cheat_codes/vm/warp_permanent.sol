// This test ensures that the block timestamp is permanent once it is changed with the warp cheatcode
interface CheatCodes {
    function warp(uint256) external;
}

contract TestContract {
    // Obtain our cheat code contract reference.
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
    uint64 timestampOffset;
    event Test(uint256);

    constructor() {
        // Set the starting timestamp
        timestampOffset = 12345;
        cheats.warp(block.timestamp + 24 hours + 1);
        emit Test(0);
    }

    function test(uint64 x) public {
        // The new timestamp should be greater than the original timestamp
        assert(block.timestamp > 12346);
    }
}

