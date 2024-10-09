// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

// This test startPrank (spoof msg.sender on subsequent calls in the current scope) and stopPrank (restore original msg.sender).
interface CheatCodes {
    function addr(uint256 privateKey) external returns (address);
    function startPrank(address) external;
    function stopPrank() external;
}

contract Ownable {
    address public owner;

    constructor() {
        owner = msg.sender;
    }
}

contract TestContract {
    TestContract thisExternal = TestContract(address(this));
    int currentDepth;
    bool calledThroughTestFunction;

    Ownable public one;
    Ownable public two;
    Ownable public three;
    Ownable public four;
    Ownable public five;
    Ownable public six;
    Ownable public seven;
    address prankAddr = address(7);

    // Obtain our cheat code contract reference.
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function prankAndGetSenderAtDepth(
        int prankDepth,
        address prankAddress,
        int senderDepth
    ) public returns (address) {
        // This can't be called directly
        require(calledThroughTestFunction);

        // This test should be written so prank depth is never less than the sender fetching depth, and the sender
        // depth must be a positive number. Prank depth can be negative, to not prank at all.
        require(senderDepth >= prankDepth);
        require(senderDepth >= 0);
        require(senderDepth < 10); // disallow printing at depth > 10, in case fuzzer hits function this directly

        // If we are at the depth we wanted to prank at, call prank.
        if (currentDepth == prankDepth) {
            // Change value and verify.
            cheats.startPrank(prankAddress);
        }

        // If we haven't reached our depth limit, return our result from a further external call.
        if (currentDepth < senderDepth) {
            currentDepth++;
            address sender = thisExternal.prankAndGetSenderAtDepth(
                prankDepth,
                prankAddress,
                senderDepth
            );
            currentDepth--;
            return sender;
        }

        // We have reached our depth limit, return the sender.
        return msg.sender;
    }

    function test() public {
        address owner1 = cheats.addr(0x123456);
        address owner2 = cheats.addr(0x234567);
        address originalMsgSender = msg.sender;
        address thisExternalAddr = address(this);

        cheats.startPrank(owner1);
        one = new Ownable();
        two = new Ownable();
        three = new Ownable();

        // Ensure that the msg.sender for this scope has not changed
        assert(msg.sender == originalMsgSender);

        // Check that the prank changed the msg.sender for subsequent external calls
        assert(one.owner() == owner1);
        assert(two.owner() == owner1);
        assert(three.owner() == owner1);

        cheats.startPrank(owner2);
        four = new Ownable();
        five = new Ownable();

        // Ensure that the pranked address has changed
        assert(four.owner() == owner2);
        assert(five.owner() == owner2);

        cheats.stopPrank();
        six = new Ownable();
        seven = new Ownable();

        // Check that the msg.sender of external calls is reset to the original
        assert(six.owner() == thisExternalAddr);
        assert(seven.owner() == thisExternalAddr);

        calledThroughTestFunction = true;

        // Check that the msg.sender for nested calls is not spoofed by startPrank
        assert(prankAndGetSenderAtDepth(1, prankAddr, 1) == thisExternalAddr);
        assert(prankAndGetSenderAtDepth(1, prankAddr, 2) == prankAddr);
        assert(prankAndGetSenderAtDepth(1, prankAddr, 3) == thisExternalAddr);

        calledThroughTestFunction = false;
    }
}
