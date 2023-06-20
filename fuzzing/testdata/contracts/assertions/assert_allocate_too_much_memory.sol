contract TestContract {
    function allocateTooMuchMemory() public {
        uint256[] memory myArray = new uint256[](2**256); // Allocate too much memory
    }
}