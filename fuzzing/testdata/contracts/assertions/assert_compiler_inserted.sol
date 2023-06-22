contract TestContract {
    function triggerCompilerPanic() public pure {
        uint nonExistentVariable;
    }
}
