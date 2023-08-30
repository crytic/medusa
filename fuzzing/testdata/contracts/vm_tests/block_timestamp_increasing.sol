contract TestContract {
    uint startingBlockTimestamp;

    constructor() public {
        // Record the block timestamp when we deploy
        startingBlockTimestamp = block.timestamp;
    }

    function waitTimestamp() public {
        // This method does nothing but is left exposed so it can be called by the fuzzer to advance blocks/timestamps.
    }

    function property_increase_block_timestamp() public view returns (bool) {
        // ASSERTION: block timestamp should never increase more than 10 (we expect failure)
        return !(block.timestamp - startingBlockTimestamp >= 10);
    }
}
