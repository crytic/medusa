contract TestBlockNumber {
    uint startingBlockNumber;

    constructor() public {
        // Record the block number when we deploy
        startingBlockNumber = block.number;
    }

    function doNothing() public {
        // This method does nothing but is left exposed so it can be called by the fuzzer to advance block.number
    }

    function fuzz_increase_block_number_by_10() public view returns (bool) {
        // ASSERTION: block number should never increase more than 10
        return !(block.number - startingBlockNumber >= 10);
    }
}
