contract TestContract {
    function popEmptyArray() public {
        uint256[] memory myArray;
        uint256 value = myArray.pop(); // Pop from empty array
    }
}