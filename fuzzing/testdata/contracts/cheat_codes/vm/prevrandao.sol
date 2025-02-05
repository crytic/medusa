// This test ensures that the block difficulty can be set with cheat codes
interface CheatCodes {
    function prevrandao(bytes32) external;
}

contract TestContract {
    function test(uint256 x) public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Change value and verify.
        cheats.prevrandao(bytes32(uint256(42)));
        assert(block.prevrandao == 42);
        cheats.prevrandao(bytes32(uint256(420)));
        assert(block.prevrandao == 420);
    }
}
