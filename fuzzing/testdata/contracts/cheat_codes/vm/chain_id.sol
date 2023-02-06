interface CheatCodes {
    function chainId(uint256) external;
}

contract TestContract {
    function getChainID() public view returns (uint256) {
        uint256 id;
        assembly {
            id := chainid()
        }
        return id;
    }

    function test(uint256 x) public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Change value and verify.
        cheats.chainId(x);
        assert(getChainID() == x);
        cheats.chainId(7);
        assert(getChainID() == 7);
    }
}
