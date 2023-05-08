// This test ensures that the msg.sender can be set with cheat codes.
// It tests prank (spoof msg.sender on next call in the same scope), and prankHere (spoof msg.sender in this scope)
interface CheatCodes {
    function prank(address) external;
    function prankHere(address) external;
}

contract TestContract {
    TestContract thisExternal = TestContract(address(this));
    int currentDepth;
    bool calledThroughTestFunction;

    function prankAndGetSenderAtDepth(bool prankHere, int prankDepth, address prankAddress, int senderDepth) public returns (address) {
        // This can't be called directly
        require(calledThroughTestFunction);

        // This test should be written so prank depth is never less than the sender fetching depth, and the sender
        // depth must be a positive number. Prank depth can be negative, to not prank at all.
        require(senderDepth >= prankDepth);
        require(senderDepth >= 0);
        require(senderDepth < 10); // disallow printing at depth > 10, in case fuzzer hits function this directly

        // If we are at the depth we wanted to prank at, call prank.
        if (currentDepth == prankDepth) {
            // Obtain our cheat code contract reference.
            CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

            // Change value and verify.
            if (!prankHere) {
                cheats.prank(prankAddress);
            } else {
                cheats.prankHere(prankAddress);
            }
        }

        // If we haven't reached our depth limit, return our result from a further external call.
        if (currentDepth < senderDepth) {
            currentDepth++;
            address sender = thisExternal.prankAndGetSenderAtDepth(prankHere, prankDepth, prankAddress, senderDepth);
            currentDepth--;
            return sender;
        }

        // We have reached our depth limit, return the sender.
        return msg.sender;
    }

    function test() public {
        // Cache some original variables
        address prankAddr = address(7);
        address originalMsgSender = msg.sender;
        address thisExternalAddr = address(this);
        assert(thisExternalAddr == address(thisExternal));

        calledThroughTestFunction = true;

        // Run some tests below. We prank at depth 0, since we make an internal call here and we don't want a
        // test that pollutes our current call scope.

        // Try prank at depth 1 (we expect only the next call at depth 2 to be pranked).
        assert(prankAndGetSenderAtDepth(false, 1, prankAddr, 1) == thisExternalAddr);
        assert(prankAndGetSenderAtDepth(false, 1, prankAddr, 2) == prankAddr);
        assert(prankAndGetSenderAtDepth(false, 1, prankAddr, 3) == thisExternalAddr);

        // Try obtaining the same without prank afterwards.
        assert(prankAndGetSenderAtDepth(false, -1, prankAddr, 0) == originalMsgSender);
        assert(prankAndGetSenderAtDepth(false, -1, prankAddr, 1) == thisExternalAddr);
        assert(prankAndGetSenderAtDepth(false, -1, prankAddr, 2) == thisExternalAddr);


        // Try prankHere at depth 1 (we expect only depth 1 to be pranked).
        assert(prankAndGetSenderAtDepth(true, 1, prankAddr, 1) == prankAddr);
        assert(prankAndGetSenderAtDepth(true, 1, prankAddr, 2) == thisExternalAddr);
        assert(prankAndGetSenderAtDepth(true, 1, prankAddr, 3) == thisExternalAddr);

        // Try obtaining the same without prankHere afterwards.
        assert(prankAndGetSenderAtDepth(true, -1, prankAddr, 0) == originalMsgSender);
        assert(prankAndGetSenderAtDepth(true, -1, prankAddr, 1) == thisExternalAddr);
        assert(prankAndGetSenderAtDepth(true, -1, prankAddr, 2) == thisExternalAddr);

        calledThroughTestFunction = false;
    }
}
