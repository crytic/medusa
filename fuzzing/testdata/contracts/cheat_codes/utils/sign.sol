interface CheatCodes {
    function sign(uint256, bytes32) external returns (uint8, bytes32, bytes32);
}

contract TestContract {
    function test() public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        bytes32 digest = keccak256("Data To Sign");

        // Call cheats.sign
        (uint8 v, bytes32 r, bytes32 s) = cheats.sign(0x6df21769a2082e03f7e21f6395561279e9a7feb846b2bf740798c794ad196e00, digest);
        address signer = ecrecover(digest, v, r, s);
        assert(signer == 0xdf8Ef652AdE0FA4790843a726164df8cf8649339);
    }
}
