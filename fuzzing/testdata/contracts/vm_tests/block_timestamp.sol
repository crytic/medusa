contract TestBlockTimestamp {
    uint startingBlockTimestamp;

    constructor() public {
        // Record the block timestamp when we deploy
        startingBlockTimestamp = block.timestamp;
    }

    function doNothing() public {
        // This method does nothing but is left exposed so it can be called by the fuzzer to advance blocks
    }

    function fuzz_increase_block_timestamp() public view returns (bool) {
        // ASSERTION: block number should never increase more than 10
        return !(block.timestamp - startingBlockTimestamp >= 10);
    }
}
