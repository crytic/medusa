// This contract ensures the fuzzer will report an error when it encounters a revert.
contract TestContract {
    function failRequire(uint value) public {
        // This should trigger a test failure due to a failing require statement (without an error message)
        require(false);
    }

    function failRequireWithErrorString(uint value) public {
        // This should trigger a test failure due to a failing require statement (with an error message)
        require(false, "Require error");
    }

    function failRevert(uint value) public {
        // This should trigger a test failure on encountering a revert instruction (without an error message)
        revert();
    }

    function failRevertWithErrorString(uint value) public {
        // This should trigger a test failure on encountering a revert instruction (with an error message)
        revert("Function reverted");
    }
}