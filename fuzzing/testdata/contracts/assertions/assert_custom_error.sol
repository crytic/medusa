// This contract ensures the fuzzer reports an assertion failure when a tested method reverts with a custom error that
// was configured to be treated as a failing case, and does not report one for custom errors that were not configured.
contract TestContract {
    error Insolvent(uint256 value);
    error Unauthorized();

    function failCustomError(uint256 value) public {
        // ASSERTION: This should trigger if `Insolvent(uint256)` is configured as a failing custom error.
        if (value % 2 == 0) {
            revert Insolvent(value);
        }
    }

    function failUnlistedCustomError(uint256 value) public {
        // This should not trigger, as `Unauthorized()` is never configured as a failing custom error.
        revert Unauthorized();
    }
}
