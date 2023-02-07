// This test ensures that the coinbase can be set with cheat codes
interface CheatCodes {
    function coinbase(address) external;
}

contract TestContract {
    function test(address x) public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Change value and verify.
        cheats.coinbase(x);
        assert(block.coinbase == x);
        cheats.coinbase(address(7));
        assert(block.coinbase == address(7));
    }
}
