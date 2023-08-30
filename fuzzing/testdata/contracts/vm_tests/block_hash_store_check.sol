// This contract stores every block hash of the previous 10 blocks that were recorded. If they have already been
// recorded in a previous call, it checks all the ones set on the next call for inconsistency. This requires this
// contract be called every block. Thus it requires a max block number delay of 1 to operate appropriately.
contract TestContract {
    mapping(uint => bytes32) hashes;
    mapping(uint => bool) hashesSet;
    bool failedTest;
    bool ranBefore;
    uint lastBlockNumber;
    function updateBlockHashes() public {
        // ASSERTION: current block hash should always be zero.
        bytes32 current = blockhash(block.number);
        if (current != bytes32(0)) {
            failedTest = true;
            return;
        }

        // ASSERTION: future block hash should always be zero.
        bytes32 future = blockhash(block.number + 1);
        if (future != bytes32(0)) {
            failedTest = true;
            return;
        }

        // Loop for the last 10 block hashes and verify they are what we previously recorded (also below)
        for(uint i = 1; i <= 10 && i <= block.number; i++) {
            // Obtain the block hash for this indexed block
            bytes32 hash = blockhash(block.number - i);

            // If it's not the immediate last block (first time we'll record it), perform verification for our test.
            if (i > 1) {
                // Ensure we're not processing the block twice in the same block, or that this isn't the first time
                // we're running it (as nothing would've been recorded).
                if (lastBlockNumber != 0 && lastBlockNumber != block.number) {
                    // If we found a hash that was set, but doesn't match, report a failure.
                    if (hashesSet[block.number - i] && hashes[block.number - i] != hash) {
                        failedTest = true;
                        return;
                    }
                }
            }

            // Store the hash
            hashes[block.number - i] = hash;
            hashesSet[block.number - i] = true;
        }

        // Set our current processed block number.
        lastBlockNumber = block.number;
    }

    function property_violate_block_hash_continuity() public view returns (bool) {
        // ASSERTION: we fail if our blockHash works as expected so our fuzzer will catch it.
        return !failedTest;
    }
}
