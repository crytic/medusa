// This test ensures that account code can be set with cheat codes
interface CheatCodes {
    function etch(address, bytes calldata) external;
}

contract TestContract {
    function test() public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Obtain our original code hash for an account.
        address acc = address(777);
        bytes32 originalCodeHash;
        assembly { originalCodeHash := extcodehash(acc) }

        // Change value and verify.
        cheats.etch(acc, address(this).code);
        bytes32 updatedCodeHash;
        assembly { updatedCodeHash := extcodehash(acc) }
        assert(originalCodeHash != updatedCodeHash);

        // Etch another value back so re-running this method will not fail.
        cheats.etch(acc, address(888).code);
    }
}
