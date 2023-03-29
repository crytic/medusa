interface CheatCodes {
    function addr(uint256) external returns (address);
}


contract TestContract {
    function test() public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        uint256 privateKey = 0x6df21769a2082e03f7e21f6395561279e9a7feb846b2bf740798c794ad196e00;
        address expectedAddress = 0xdf8Ef652AdE0FA4790843a726164df8cf8649339;

        // Call cheats.addr
        address result = cheats.addr(privateKey);
        assert(result == expectedAddress);
    }
}
