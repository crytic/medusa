// This contract ensures the fuzzer will not report an assertion error when performing other reverting operations.
contract TestContract {
    function failRequire(uint value) public {
        // This should not trigger, as it's not an assertion.
        require(false);
    }

    function failRevert(uint value) public {
        // This should not trigger, as it's not an assertion.
        revert();
    }
}
