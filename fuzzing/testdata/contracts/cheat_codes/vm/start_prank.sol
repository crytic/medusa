// This test ensures that the msg.sender can be set with cheat codes.
// It tests startPrank/endPrank (spoof msg.sender on subsequent calls between start and stop)
interface CheatCodes {
    function startPrank(address msgSender) external;
    function stopPrank() external;
}

contract TestContract {
    bool calledThroughTestFunction; // prevents fuzzer directly accessing startStopPrankAtDepthAndGetSender
    TestContract thisExternal = TestContract(address(this));
    CheatCodes vm = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function oneStartPrankDeep(address originalMsgSender) external
    {
        // This can't be called directly
        require(calledThroughTestFunction);

        assert(msg.sender == address(1)); // from main entry point

        vm.startPrank(address(2));
        assert(msg.sender == address(1)); // from main entry point

        thisExternal.twoStartPrankDeep(originalMsgSender);
        assert(msg.sender == address(1)); // from main entry point

        thisExternal.twoStartPrankDeep(originalMsgSender);
        assert(msg.sender == address(1)); // from main entry point

        vm.stopPrank();
        assert(msg.sender == address(1)); // from main entry point

        // stopPrank does not act like a stack allowing you to do multiple start/stop pranks
        // encapsulated within each other. Although the current call scope still had its
        // msg.sender set, a subsequent call should not. Lets test this.
        thisExternal.checkStartPrankInner(originalMsgSender, false);
    }

    function twoStartPrankDeep(address originalMsgSender) external
    {
        // This can't be called directly
        require(calledThroughTestFunction);

        assert(msg.sender == address(2)); // from oneStartPrankDeep

        thisExternal.threeStartPrankDeep(originalMsgSender);
    }

    function threeStartPrankDeep(address originalMsgSender) external
    {
        // This can't be called directly
        require(calledThroughTestFunction);

        assert(msg.sender == address(thisExternal)); // startPrank only works one level below invocation, not two.
    }

    function checkStartPrankInner(address value, bool equal) external
    {
        // This can't be called directly
        require(calledThroughTestFunction);
        if (equal)
        {
            assert(msg.sender == value);
        }
        else
        {
            assert(msg.sender != value);
        }
    }

    function tryFinalInnerStartPrank() external
    {
        vm.startPrank(address(4));
    }

    function test_startPrank() public {
        // Cache some original variables
        address originalMsgSender = msg.sender;
        address thisExternalAddr = address(this);
        assert(thisExternalAddr == address(thisExternal));

        calledThroughTestFunction = true;

        // Run a test doing startPrank, call, startPrank, call, exit, stopPrank, exit, stopPrank
        // (an inner startPrank within an existing one, with associated stops)

        vm.startPrank(address(1));
        assert(msg.sender == originalMsgSender);

        thisExternal.oneStartPrankDeep(originalMsgSender);
        assert(msg.sender == originalMsgSender);

        thisExternal.checkStartPrankInner(originalMsgSender, false);
        assert(msg.sender == originalMsgSender);

        vm.stopPrank();
        assert(msg.sender == originalMsgSender);

        thisExternal.checkStartPrankInner(originalMsgSender, false);
        assert(msg.sender == originalMsgSender);

        thisExternal.tryFinalInnerStartPrank();
        thisExternal.checkStartPrankInner(thisExternalAddr, true);
        vm.stopPrank();
        vm.stopPrank(); // more stopPrank than startPrank calls should not fail

        calledThroughTestFunction = false;
    }
}
