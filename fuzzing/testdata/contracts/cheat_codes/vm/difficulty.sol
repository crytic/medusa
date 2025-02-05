// This test ensures that the difficulty cheatcode is a no-op
interface CheatCodes {
    function difficulty(uint256) external;
}

contract TestContract {
    function test(uint256 x) public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Use try catch
        uint256 originalDifficulty = block.difficulty;
        // Update the difficulty
        cheats.difficulty(x);
        // Make sure that the new difficulty is the same as the original
        assert(block.difficulty == originalDifficulty);
    }
}
