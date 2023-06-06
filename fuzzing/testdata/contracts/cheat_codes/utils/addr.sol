interface CheatCodes {
    function addr(uint256) external returns (address);
}


contract TestContract {
    function test() public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Test with random private key
        uint256 pkOne = 0x6df21769a2082e03f7e21f6395561279e9a7feb846b2bf740798c794ad196e00;
        address addrOne = 0xdf8Ef652AdE0FA4790843a726164df8cf8649339;
        address result = cheats.addr(pkOne);
        assert(result == addrOne);

        // Test with private key that requires padding
        uint256 pkTwo = 1;
        address addrTwo = 0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf;
        result = cheats.addr(pkTwo);
        assert(result == addrTwo);

        // Test with zero
        uint256 pkThree = 0;
        cheats.addr(pkThree);
        // A private key of zero is not allowed so if we hit this assertion, then cheats.addr() did not revert which
        // is incorrect
        assert(false);
    }
}
