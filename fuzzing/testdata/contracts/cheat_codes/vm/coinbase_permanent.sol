// This test ensures that the coinbase is permanent once it is changed with the coinbase cheatcode
interface CheatCodes {
    function coinbase(address) external;
}

contract TestContract {
    // Obtain our cheat code contract reference.
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
    address newCoinbase = address(42);

    constructor() {
        // Update the coinbase
        cheats.coinbase(newCoinbase);
    }

    function test(address x) public {
        // Ensure that the coinbase is set permanently
        assert(block.coinbase == newCoinbase);
    }
}
