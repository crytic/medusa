// This test ensures that we can take a snapshot of the current state of the testchain and revert to the state at that snapshot using the snapshot and revertTo cheatcodes
pragma solidity ^0.8.0;

interface CheatCodes {
    function warp(uint256) external;

    function deal(address, uint256) external;

    function snapshot() external returns (uint256);

    function revertTo(uint256) external returns (bool);
}

struct Storage {
    uint slot0;
    uint slot1;
}

contract TestContract {
    Storage store;
    uint256 timestamp;

    function test() public {
        // Obtain our cheat code contract reference.
        CheatCodes cheats = CheatCodes(
            0x7109709ECfa91a80626fF3989D68f67F5b1DD12D
        );

        store.slot0 = 10;
        store.slot1 = 20;
        timestamp = block.timestamp;
        cheats.deal(address(this), 5 ether);

        // Save state
        uint256 snapshot = cheats.snapshot();

        // Change state
        store.slot0 = 300;
        store.slot1 = 400;
        cheats.deal(address(this), 500 ether);
        cheats.warp(12345);

        // Assert that state has been changed
        assert(store.slot0 == 300);
        assert(store.slot1 == 400);
        assert(address(this).balance == 500 ether);
        assert(block.timestamp == 12345);

        // Revert to snapshot
        cheats.revertTo(snapshot);

        // Ensure state has been reset
        assert(store.slot0 == 10);
        assert(store.slot1 == 20);
        assert(address(this).balance == 5 ether);
        assert(block.timestamp == timestamp);
    }
}
