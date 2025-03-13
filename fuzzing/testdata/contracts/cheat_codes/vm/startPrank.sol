// This test ensures that the msg.sender can be set with cheat codes.
// It tests startPrank/endPrank (spoof msg.sender on subsequent calls between start and stop)
interface CheatCodes {
    function startPrank(address msgSender) external;
    function startPrank(address msgSender, bool delegateCall) external;
    function startPrank(address msgSender, address txOrigin) external;
    function startPrank(address msgSender, address txOrigin, bool delegateCall) external;

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
        //assert(tx.origin == address(11));

        vm.startPrank(address(2), address(22));
        assert(msg.sender == address(1)); // from main entry point
        //assert(tx.origin == address(11));

        thisExternal.twoStartPrankDeep(originalMsgSender);
        assert(msg.sender == address(1)); // from main entry point
        //assert(tx.origin == address(11));

        thisExternal.twoStartPrankDeep(originalMsgSender);
        assert(msg.sender == address(1)); // from main entry point
        //assert(tx.origin == address(11));

        vm.stopPrank();
        assert(msg.sender == address(1)); // from main entry point
        //assert(tx.origin == address(11));

        // stopPrank does not act like a stack allowing you to do multiple start/stop pranks
        // encapsulated within eachother. Although the current call scope still had its
        // msg.sender set, a subsequent call should not. Lets test this.
        thisExternal.checkStartPrankInner(originalMsgSender, false);
    }

    function twoStartPrankDeep(address originalMsgSender) external
    {
        // This can't be called directly
        require(calledThroughTestFunction);

        assert(msg.sender == address(2)); // from oneStartPrankDeep
        //assert(tx.origin == address(22));

        thisExternal.threeStartPrankDeep(originalMsgSender);
    }

    function threeStartPrankDeep(address originalMsgSender) external
    {
        // This can't be called directly
        require(calledThroughTestFunction);

        assert(msg.sender == address(thisExternal)); // startPrank only works one level below invocation, not two.
        //assert(tx.origin == address(22)); // tx.origin is tx-wide, shared across scopes
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

        // Note: We should be testing tx.origin here too, but there is a bug in Foundry!
        // We'll keep this test contract compatible across common solutions.
    }


    /// -----------------------------------------------------------------------------------------------

    function startStopPrankAtAndTest(address originalMsgSender, int currentDepth, int startPrankDepth, int stopPrankDepth) external
    {
        // This can't be called directly
        require(calledThroughTestFunction);

        if (currentDepth == startPrankDepth)
        {
            vm.startPrank(address(3), address(33));
        }

        if (currentDepth == startPrankDepth + 1)
        {
            return;
        }

        thisExternal.startStopPrankAtAndTest(originalMsgSender, currentDepth + 1, startPrankDepth, stopPrankDepth);

        // If we started pranking and didn't hit stop yet, stop.
        if ((stopPrankDepth <= startPrankDepth && currentDepth <= startPrankDepth && currentDepth >= stopPrankDepth) ||
        (stopPrankDepth > startPrankDepth && currentDepth >= startPrankDepth && currentDepth <= stopPrankDepth))
        {
            thisExternal.checkStartPrankInner(address(3), true);
        }

        if (currentDepth == stopPrankDepth)
        {
            vm.stopPrank();
        }
    }

    function testFinalInnerStartPrank() external
    {
        vm.startPrank(address(4), address(44));
    }

    function test_startPrank() public {
        // Cache some original variables
        address originalMsgSender = msg.sender;
        address originalTxOrigin = tx.origin;
        address thisExternalAddr = address(this);
        assert(thisExternalAddr == address(thisExternal));

        calledThroughTestFunction = true;

        // Run a test doing startPrank, call, startPrank, call, exit, stopPrank, exit, stopPrank
        // (an inner startPrank within an existing one, with associated stops)

        vm.startPrank(address(1), address(11));
        assert(msg.sender == originalMsgSender);
        //assert(tx.origin == originalTxOrigin);

        thisExternal.oneStartPrankDeep(originalMsgSender);
        assert(msg.sender == originalMsgSender);
        //assert(tx.origin == address(11));

        thisExternal.checkStartPrankInner(originalMsgSender, false);
        assert(msg.sender == originalMsgSender);
        //assert(tx.origin == address(11));

        vm.stopPrank();
        assert(msg.sender == originalMsgSender);
        //assert(tx.origin == address(11));

        thisExternal.checkStartPrankInner(originalMsgSender, false);
        assert(msg.sender == originalMsgSender);
        //assert(tx.origin == address(11));

        // Run a test
        //thisExternal.startStopPrankAtAndTest(originalMsgSender, 0, 5, 5);
        //thisExternal.startStopPrankAtAndTest(originalMsgSender, 0, 5, 4);

        thisExternal.testFinalInnerStartPrank();
        thisExternal.checkStartPrankInner(thisExternalAddr, true);
        vm.stopPrank();
        vm.stopPrank(); // more stopPrank than startPrank calls should not fail

        calledThroughTestFunction = false;
    }
}