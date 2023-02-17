interface CheatCodes {
    function addr(uint256) external returns (address);
}

// Source for test case: https://book.getfoundry.sh/cheatcodes/addr
contract TestContract {
    function test() public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        uint256 privateKey = 1;
        address expectedAddress = 0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf;

        // Call cheats.addr
        address result = cheats.addr(privateKey);
        assert(result == expectedAddress);
    }
}
