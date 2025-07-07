// This test ensures that the expectRevert works as expected
interface CheatCodes {
    function expectRevert() external;
    function prank(address msgSender) external;
}
contract Target{
    function good() public {}
}
interface FakeTarget{
    function good() external;
    function bad() external;
}
contract TestContract {
    
    function test_true() public {
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
        FakeTarget target = FakeTarget(address(new Target()));
        cheats.expectRevert();
        target.bad();
        
        // this always happens (only useful if tested with manual coverage check)
        assert(true);
    }
    function test_false() public {
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
        FakeTarget target = FakeTarget(address(new Target()));
        cheats.expectRevert();
        target.good();

        // this never happens (only useful if tested with manual coverage check)
        assert(false);
    }

    function test_true_with_prank() public {
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
        FakeTarget target = FakeTarget(address(new Target()));
        cheats.expectRevert();
        // Calls to the VM are ignored from the expectRevert logic
        // So this test that
	    cheats.prank(address(0x41414141));
        target.bad();

        // this always happens (only useful if tested with manual coverage check)
        assert(true);
    }

        
    function test_false_with_prank() public {
        CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);
        FakeTarget target = FakeTarget(address(new Target()));
        cheats.expectRevert();
        // Calls to the VM are ignored from the expectRevert logic
        // So this test that
	    cheats.prank(address(0x41414141));
        target.good();

        // this never happens (only useful if tested with manual coverage check)
        assert(false);
    }
}