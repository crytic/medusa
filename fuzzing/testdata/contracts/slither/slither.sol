// This test ensures that all the constants in this contract are added to the value set
contract TestContract {
    // Uint constant
    uint256 constant x = 123 + 12;

    constructor() {}

    function testFuzz(uint256 z) public {
        // Set z to x so that 123, 12, and (123 + 12 = 135) are captured as constants
        z = x;
        // Add a bunch of other constants
        int256 y = 456;
        address addr = address(0);
        bool b = true;
        string memory str = "Hello World!";
        assert(false);
    }
}
