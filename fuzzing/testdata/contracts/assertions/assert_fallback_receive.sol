// This contract ensures the fuzzer can call both fallback and receive functions.
// When both are called at least once, an assertion will fail.
contract TestContract {
    bool fallbackCalled = false;
    bool receiveCalled = false;
    uint256 x;

    // Fallback function - sets fallbackCalled to true
    // Can receive ETH and arbitrary calldata
    fallback() external payable {
        fallbackCalled = true;
    }

    // Receive function - sets receiveCalled to true
    // Only called with empty calldata and ETH value
    receive() external payable {
        receiveCalled = true;
    }

    // Test function that fails when both fallback and receive have been called
    function testFunc(uint256 _x) public {
        x = _x;
        // ASSERTION: Both fallback and receive should not have been called
        assert(!(fallbackCalled && receiveCalled));
    }
}
