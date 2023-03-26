interface CheatCodes {
    function toString(bool) external returns (string memory);
}

contract TestContract {
    CheatCodes cheats;

    constructor() {
        cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
    }

    function testPrecompileAndFail() public {
        bool test = true;
        string memory expectedString = "true";

        // Call cheats.toString
        string memory result = cheats.toString(test);
        assert(keccak256(abi.encodePacked(result)) == keccak256(abi.encodePacked(expectedString)));

        // Fail test immediately.
        assert(false);
    }
}
