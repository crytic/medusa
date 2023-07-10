// This contract attempts to allocate an excessive amount of memory by creating an array with a length of 2^64 causing a panic.
// PanicCodeAllocateTooMuchMemory = 0x41

contract TestContract {
    function allocateTooMuchMemory() public {
        uint256[] memory myArray = new uint256[](2**64); // Allocate too much memory
        myArray[2**64 - 1] = 42;
    }
}
