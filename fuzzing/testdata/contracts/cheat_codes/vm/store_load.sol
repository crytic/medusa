// This test ensures that account storage can be get and set with cheat codes
interface CheatCodes {
    function load(address, bytes32) external returns (bytes32);
    function store(address, bytes32, bytes32) external;
}

contract TestContract {
    uint x = 123;
    uint y = 0;
    function test() public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

        // Load and verify x
        bytes32 value = cheats.load(address(this), bytes32(uint(0)));
        assert(value == bytes32(uint(123)));

        // Change y, load it, verify it.
        y = 321;
        value = cheats.load(address(this), bytes32(uint(1)));
        assert(value == bytes32(uint(321)));

        // Store into y, verify it.
        cheats.store(address(this), bytes32(uint(1)), bytes32(uint(456)));
        assert(y == 456);
    }
}
