// This test ensures that label can be set for an address
interface CheatCodes {
    function label(address, string memory) external;
}

contract LabelContract {
    function testLabel() public {
        assert(false);
    }
}

contract TestContract {

    function test() public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Create a contract
        LabelContract alice = new LabelContract();

        // set label and verify.
        cheats.label(address(alice), "Alice");
        alice.testLabel();
    }
}
