// An attempt to call an uninitialized function pointer would cause a panic
// PanicCodeCallUninitializedVariable = 0x51

contract TestContract {

    function uninitializedVariableCall() public returns (int)
    {
        // Variable containing a function pointer
        function (int, int) internal pure returns (int) funcPtr;

        // This call will fail because funcPtr is still a zero-initialized function pointer
        return funcPtr(4, 5);
    }

}
