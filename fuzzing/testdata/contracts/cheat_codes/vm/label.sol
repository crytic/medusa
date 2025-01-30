// This test ensures that label can be set for an address
interface CheatCodes {
    function label(address, string memory) external;
}

contract LabelContract {
    function testLabel() public {
        // Throw an assertion failure so that we can capture the execution trace
        assert(false);
    }
}

contract TestContract {
    // Obtain our cheat code contract reference.
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
    // Create a contract
    LabelContract alice = new LabelContract();

    constructor() {
        // Set label for LabelContract to "Alice"
        cheats.label(address(alice), "Alice");
    }

    function test() public {
        // Call the label contract
        alice.testLabel();
    }
}
