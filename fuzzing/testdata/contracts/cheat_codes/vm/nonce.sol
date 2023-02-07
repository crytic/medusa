// This test ensures that account nonces can be get and set with cheat codes
interface CheatCodes {
    function getNonce(address) external returns (uint64);
    function setNonce(address, uint64) external;
}

contract TestContract {
    function test(uint64 x) public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Change value and verify.
        address acc = address(msg.sender);
        cheats.setNonce(acc, x);
        assert(cheats.getNonce(acc) == x);
        cheats.setNonce(acc, 7);
        assert(cheats.getNonce(acc) == 7);
    }
}
