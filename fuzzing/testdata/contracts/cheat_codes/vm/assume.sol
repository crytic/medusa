// This test ensures that the chainId can be set with cheat codes
interface CheatCodes {
    function assume(bool) external;
}

contract TestContract {
    
    function test_true() public {
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
        cheats.assume(true);
        // this always happens (only useful if tested with manual coverage check)
        assert(true);
    }

    function test_false() public {
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
        cheats.assume(false);
        // this is not reachable
        assert(false);
    }
}