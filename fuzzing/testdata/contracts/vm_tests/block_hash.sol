contract TestBlockHash {
    mapping(uint => bytes32) hashes;
    bool failedTest;
    bool ranBefore;
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
                if (ranBefore && hashes[block.number - i] != hash) {
                    failedTest = true;
                    return;
                }
            }

            // Store the hash
            hashes[block.number - i] = hash;
        }
        ranBefore = true;
    }

    function fuzz_violate_block_hash_continuity() public view returns (bool) {
        // ASSERTION: we fail if our blockHash works as expected so our fuzzer will catch it.
        return !failedTest;
    }
}
