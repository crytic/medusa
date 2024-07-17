// This contract ensures the fuzzer can encounter an immediate assertion failure.
contract TestContract {
    function callingMeFails(uint value) public {
        // ASSERTION: We always fail when you call this function.
        assert(false);
    }
}
