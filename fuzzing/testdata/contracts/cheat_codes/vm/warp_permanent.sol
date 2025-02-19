// This test ensures that the block timestamp is permanent once it is changed with the warp cheatcode
interface CheatCodes {
    function warp(uint256) external;
}

contract TestContract {
    // Obtain our cheat code contract reference.
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
    uint64 timestampOffset;

    constructor() {
        // Set the starting timestamp
        timestampOffset = 12345;
        cheats.warp(block.timestamp + 12345);
    }

    function test(uint64 x) public {
        // We know that the block timestamp originally will be 1 so we need
        // to make sure that the new block timestamp is 1 more than the offset
        assert(block.timestamp >= timestampOffset + 1);
    }
}

