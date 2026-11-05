// This contract tests basic sometimes assertion functionality.
// The fuzzer should discover that sometimes_counterIncremented passes
// and sometimes_impossibleCondition fails.
contract TestContract {
    uint256 public counter;

    function increment(uint256 x) public {
        if (x % 2 == 0) {
            counter++;
        }
    }

    // This should PASS - counter increments ~50% of the time
    function sometimes_counterIncremented() public view {
        require(counter > 0, "Counter should be incremented sometimes");
    }

    // This should FAIL - impossible condition
    function sometimes_impossibleCondition() public view {
        require(counter > 1000000, "This should never happen");
    }

    // This should PASS - always succeeds
    function sometimes_alwaysSucceeds() public pure {
        // Never reverts
    }
}
