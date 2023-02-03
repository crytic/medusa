interface CheatCodes {
    function fee(uint256) external;
}

contract TestContract {
    function test(uint256 x) public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Back up the original value.
        uint original = block.basefee;

        // Change to the provided value and verify.
        cheats.fee(x);
        assert(block.basefee == x);

        // Change back to the original value and verify.
        cheats.fee(original);
        assert(block.basefee == original);
    }
}
