// SPDX-License-Identifier: MIT
// This test verifies that shrinking can remove reverting calls while preserving cheatcode state.
// The property test depends on vm.roll and vm.warp setting specific block/timestamp values.
// A reverting call between cheatcodes and the property check should be removable during shrinking.

interface CheatCodes {
    function roll(uint256) external;
    function warp(uint256) external;
}

contract TestContract {
    // Obtain our cheat code contract reference.
    CheatCodes cheats = CheatCodes(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    // Track if cheatcodes were set up correctly
    uint256 public targetBlockNumber = 12345;
    uint256 public targetTimestamp = 67890;

    // A function that always reverts - its state changes don't matter
    function alwaysReverts() public pure {
        revert("This call always reverts");
    }

    // Set up the chain state via cheatcodes
    function setupCheatcodeState() public {
        cheats.roll(targetBlockNumber);
        cheats.warp(targetTimestamp);
    }

    // Property test that fails when block number and timestamp match targets
    // The failure depends on cheatcode state, not on any reverting call
    function property_cheatcode_state_preserved() public view returns (bool) {
        // This property fails when both conditions are met
        // The shrunk sequence should only need setupCheatcodeState() or direct cheatcode calls
        // to trigger this failure, not any reverting calls
        if (block.number == targetBlockNumber && block.timestamp == targetTimestamp) {
            return false; // Property violated - test fails
        }
        return true;
    }

    // A function that may or may not revert based on input
    function maybeReverts(uint256 x) public pure {
        if (x > 100) {
            revert("Reverted because x > 100");
        }
    }

    // Combined test function that sets up state, calls reverting function, then checks property
    function fuzz_combined_test(uint256 x) public {
        // Set up cheatcode state
        cheats.roll(targetBlockNumber);
        cheats.warp(targetTimestamp);

        // This reverting call should be removable during shrinking
        // because its state changes are rolled back and don't affect the property
        if (x > 50) {
            try this.alwaysReverts() {} catch {}
        }

        // The assertion depends only on the cheatcode state, not on the reverting call
        assert(block.number != targetBlockNumber || block.timestamp != targetTimestamp);
    }
}
