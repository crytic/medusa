// This test ensures that the block timestamp can be set with cheat codes
interface CheatCodes {
    function warp(uint256) external;
}

contract TestContract {
    // Obtain our cheat code contract reference.
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
    uint64 startingTimestamp;

    constructor() {
        // Set the starting timestamp
        startingTimestamp = 12345;
        cheats.warp(startingTimestamp);
    }

    function test(uint64 x) public {
        assert(block.timestamp > startingTimestamp);
    }
}

