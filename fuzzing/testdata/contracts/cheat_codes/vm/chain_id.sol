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

        // Back up the original value.
        uint original = getChainID();

        // Change to the provided value and verify.
        cheats.chainId(x);
        assert(getChainID() == x);

        // Change back to the original value and verify.
        cheats.chainId(original);
        assert(getChainID() == original);
    }
}
