// This test ensures that the block number is permanent once it is changed with the roll cheatcode
interface CheatCodes {
    function roll(uint256) external;
}

contract TestContract {
    // Obtain our cheat code contract reference.
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
    uint64 blockNumberOffset;

    constructor() {
        // Set the starting block number
        blockNumberOffset = 12345;
        cheats.roll(block.number + blockNumberOffset);
    }
    function test(uint256 x) public {
        // We know that the block number originally will be 1 so we need
        // to make sure that the new block number is 1 more than the offset
        assert(block.number >= blockNumberOffset + 1);
    }
}
