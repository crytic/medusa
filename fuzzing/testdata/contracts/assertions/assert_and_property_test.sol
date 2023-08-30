// This contract ensures the fuzzer can run property and assertion testing in parallel and catch both failures.
contract TestContract {
    function failing_assert_method(uint value) public {
        // ASSERTION: We always fail when you call this function.
        assert(false);
    }

    function property_failing_property() public view returns (bool) {
        // ASSERTION: fail immediately.
        return false;
    }
}
