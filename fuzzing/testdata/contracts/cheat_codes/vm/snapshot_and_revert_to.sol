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

        // Create first snapshot
        uint256 snapshot1 = cheats.snapshot();

        // Change state
        store.slot0 = 300;
        store.slot1 = 400;

        // Assert that state has been changed
        assert(store.slot0 == 300);
        assert(store.slot1 == 400);

        // Create second snapshot
        uint256 snapshot2 = cheats.snapshot();

        // Change state again
        store.slot0 = 250;
        store.slot1 = 67;

        // Assert that state has been changed
        assert(store.slot0 == 250);
        assert(store.slot1 == 67);

        // Create third snapshot
        uint256 snapshot3 = cheats.snapshot();

        // Change state again
        store.slot0 = 347;
        store.slot1 = 3;

        // Assert that state has been changed
        assert(store.slot0 == 347);
        assert(store.slot1 == 3);

        // Revert to third snapshot
        cheats.revertTo(snapshot3);

        // Ensure state has been reset to third snapshot
        assert(store.slot0 == 250);
        assert(store.slot1 == 67);

        // Revert to second snapshot
        cheats.revertTo(snapshot2);

        // Ensure state has been reset to second snapshot
        assert(store.slot0 == 300);
        assert(store.slot1 == 400);

        // Revert to first snapshot
        cheats.revertTo(snapshot1);

        // Ensure state has been reset to original
        assert(store.slot0 == 10);
        assert(store.slot1 == 20);
    }
}
