contract TestContract {
    function outOfBoundsArrayAccess() public {
        uint256[] memory myArray = new uint256[](5);
        uint256 value = myArray[6]; // Out of bounds array access
    }
}