// This test ensures that the block number is permanent once it is changed with the roll cheatcode
interface CheatCodes {
    function roll(uint256) external;
}

contract TestContract {
    // Obtain our cheat code contract reference.
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
    uint64 startingBlockNumber;

    constructor() {
        // Set the starting block number
        startingBlockNumber = 12345;
        cheats.roll(startingBlockNumber);
    }
    function test(uint256 x) public {
        assert(block.number > startingBlockNumber);
    }
}
