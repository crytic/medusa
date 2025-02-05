// This test ensures that the difficulty cheatcode will revert
interface CheatCodes {
    function difficulty(uint256) external;
}

contract TestContract {
    function test(uint256 x) public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Use try catch
        try cheats.difficulty(x) {
            // The call to difficulty should not work
            assert(false);
        } catch (bytes memory){}
    }
}
