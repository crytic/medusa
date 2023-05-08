// This test ensures that the chainId can be set with cheat codes
interface CheatCodes {
    function chainId(uint256) external;
}

contract TestContract {
    uint callCount;
    uint firstChainId;
    function getChainID() private view returns (uint256) {
        uint256 id;
        assembly {
            id := chainid()
        }
        return id;
    }

    function test(uint256 x) public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Back up our original chain id to ensure its only that way the first time we execute.
        if (callCount == 0) {
            firstChainId = getChainID();
            assert(firstChainId != 777123);
            assert(firstChainId != 888321);
        } else {
            assert(firstChainId != getChainID());
        }

        // Change value and verify.
        cheats.chainId(777123);
        assert(getChainID() == 777123);
        cheats.chainId(888321);
        assert(getChainID() == 888321);
        callCount++;
    }
}
