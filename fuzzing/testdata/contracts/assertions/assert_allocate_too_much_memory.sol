contract TestContract {
    function allocateTooMuchMemory() public {
        uint256[] memory myArray = new uint256[](2**64); // Allocate too much memory
        myArray[2**64 - 1] = 42;
    }
}