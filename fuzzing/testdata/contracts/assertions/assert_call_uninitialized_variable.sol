contract TestContract {
    uint256 uninitializedVariable;

    function callUninitializedVariable() public {
        uninitializedVariable = 42;
        address(uninitializedVariable).call(""); // Call uninitialized variable
    }
}