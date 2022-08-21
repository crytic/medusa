contract TestAssertionImmediate {
    function callingMeFails(uint value) public {
        // ASSERTION: We always fail when you call this function.
        assert(false);
    }

    function fuzz_not_solvable() public view returns (bool) {
        // ASSERTION: We put a dummy assertion here that will never fail.
        return true;
    }
}
