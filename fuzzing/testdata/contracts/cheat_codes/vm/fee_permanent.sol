// This test ensures that the base fee is permanent once it is changed with the fee cheatcode
interface CheatCodes {
    function fee(uint256) external;
}

contract TestContract {
    // Obtain our cheat code contract reference.
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
    uint256 newBaseFee = 42 gwei;

    constructor() {
        // Set the new base fee
        cheats.fee(newBaseFee);
    }

    function test(uint256 x) public {
        // Assert that the change to fee is permanent
        assert(block.basefee == newBaseFee);
    }
}
