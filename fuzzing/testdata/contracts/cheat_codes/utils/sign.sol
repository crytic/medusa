interface CheatCodes {
    function sign(uint256, bytes32) external returns (uint8, bytes32, bytes32);
}

// Source for test case: https://book.getfoundry.sh/cheatcodes/sign
contract TestContract {
    function test() public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        bytes32 digest = keccak256("Signed by Alice");
        address expectedSigner = 0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf;

        // Call cheats.sign
        (uint8 v, bytes32 r, bytes32 s) = cheats.sign(1, digest);
        address signer = ecrecover(digest, v, r, s);
        assert(signer == expectedSigner);
    }
}
